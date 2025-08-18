package main

import (
	"net"
	"sync"
	"time"

	"google.golang.org/grpc/balancer"
)

/*

Полное условие: Реализуйте балансировщик для gRPC с health-check'ами.

Требования:

 - Round-robin выбор при здоровых нодах
 - Circuit breaker при ошибках
 - Экспоненциальный backoff для health-check'ов
 - Поддержка ResolveNow
 - Метрики состояния нод

Тесты:

 - Поведение при падении ноды
 - Распределение запросов
 - Восстановление после failure

*/

type nodeState int

const (
	Open = iota
	HalfOpen
	Closed
)

type BalancerConfig struct {
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
	MaxFails            int
}

type Node struct {
	state          nodeState
	address        string
	amountOfErrors int         // количество ошибок
	rate           int         // количество запросов
	maxRates       int         // максимальное количество запросов, которое можно отправить на хост - паттерн circuit breaker
	timer          time.Ticker // тикер, который будет работать, когда circuit breaker будет в закрытом состоянии
}

type customBalancer struct {
	name     string
	cfg      BalancerConfig
	nodes    []Node
	next     int
	stopChan chan struct{}
	mu       *sync.RWMutex
	wg       *sync.WaitGroup
}

func NewBuilder(cfg BalancerConfig) balancer.Builder {
	return customBalancer{cfg: cfg, name: "customGRPCBalancer"}
}

func (b customBalancer) Name() string {
	return b.name
}

func (b customBalancer) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	stopChan := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.RWMutex
	nodes := make([]Node, 0)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopChan:
				return
			default:
				mu.RLock()
				nodesCopy := make([]Node, 0, len(nodes))
				for _, backend := range nodes {
					nodesCopy = append(nodesCopy, backend)
				}
				hcInt := b.cfg.HealthCheckInterval
				hcTime := b.cfg.HealthCheckTimeout
				maxFails := b.cfg.MaxFails
				mu.RUnlock()
				for key := range nodesCopy {
					mu.RLock()
					nodeState := nodes[key].state
					nodeAddress := nodes[key].address
					mu.RUnlock()
					if nodeState == Open {
						select {
						case <-nodes[key].timer.C:
							wg.Add(1)
							go func() {
								defer wg.Done()
								retries := 0
								hc := HealthCheckerEx{timeout: hcTime}
								timerInt := time.NewTicker(hcInt)
								defer timerInt.Stop()
								for {
									select {
									case <-stopChan:
										mu.Lock()
										nodes[key].timer.Stop()
										mu.Unlock()
										return
									case <-timerInt.C:
										state, err := hc.Check(nodeAddress)
										if err == nil {
											mu.Lock()
											if nodes[key].rate/2 > nodes[key].amountOfErrors {
												nodes[key].state = HalfOpen
												nodes[key].maxRates = 20
											} else {
												nodes[key].state = Closed
												nodes[key].maxRates = -1
											}
											nodes[key].timer.Stop()
											mu.Unlock()
											return
										}
										if err != nil || !state {
											retries++
											b.mu.Lock()
											nodes[key].amountOfErrors++
											nodes[key].rate++
											b.mu.Unlock()
											if maxFails == retries {
												b.mu.Lock()
												nodes[key].state = Open
												nodes[key].timer.Reset(10 * time.Second)
												b.mu.Unlock()
												return
											}
										}
										hcInt += 1
										timerInt.Reset(hcInt)
									}
								}
							}()
						default:
						}
					}
				}
			}
		}
	}()

	return customBalancer{
		name:     b.name,
		cfg:      b.cfg,
		nodes:    nodes,
		wg:       &wg,
		mu:       &mu,
		stopChan: stopChan,
		next:     0,
	}
}

func (b customBalancer) ExitIdle() {

}

// ResolverError вызывается gRPC, когда resolver сообщает об ошибке. Пример ошибки: проблема с резолвингом адресов.
// При этом необходимо пройтись по всем адресам и позапускать health check-и, чтобы проверить, что бэкенды доступны и работают.
func (b customBalancer) ResolverError(err error) {

	b.mu.RLock()
	nodesCopy := make([]Node, 0, len(b.nodes))
	for _, backend := range b.nodes {
		nodesCopy = append(nodesCopy, backend)
	}
	b.mu.RUnlock()

	for key := range nodesCopy {
		b.mu.RLock()
		nodeState := b.nodes[key].state
		nodeAddress := b.nodes[key].address
		b.mu.RUnlock()
		if nodeState == Open {
			b.wg.Add(1)
			go func() {
				defer b.wg.Done()
				retries := 0

				b.mu.RLock()
				hcInt := b.cfg.HealthCheckInterval
				hcTimeout := b.cfg.HealthCheckTimeout
				maxRetries := b.cfg.MaxFails
				b.mu.RUnlock()

				timerInt := time.NewTicker(hcInt)
				hc := HealthCheckerEx{timeout: hcTimeout}

				defer timerInt.Stop()

				for {
					select {
					case <-timerInt.C:
						state, err := hc.Check(nodeAddress)
						if err == nil {
							b.mu.Lock()
							if b.nodes[key].rate/2 > b.nodes[key].amountOfErrors {
								b.nodes[key].state = HalfOpen
								b.nodes[key].maxRates = 20
							} else {
								b.nodes[key].state = Closed
								b.nodes[key].maxRates = -1
							}
							b.nodes[key].rate = 0
							b.nodes[key].amountOfErrors = 0
							b.nodes[key].timer.Stop()
							b.mu.Unlock()
							return
						}
						if err != nil || !state {
							retries++
							b.mu.Lock()
							b.nodes[key].amountOfErrors++
							b.nodes[key].rate++
							b.mu.Unlock()
							if maxRetries == retries {
								b.mu.Lock()
								b.nodes[key].rate = 0
								b.nodes[key].amountOfErrors = 0
								b.nodes[key].state = Open
								b.nodes[key].timer = *time.NewTicker(10 * time.Second)
								b.mu.Unlock()
								return
							}
						}
						hcInt += 1
						timerInt.Reset(hcInt)
					case <-b.stopChan:
						return
					}
				}

			}()
		}
	}
}

// Вызывается gRPC при изменении состояния ClientConn, например,
// когда resolver предоставляет новые адреса бэкендов или обновляется конфигурация балансировки.
func (b customBalancer) UpdateClientConnState(resolver balancer.ClientConnState) error {

	var wg sync.WaitGroup
	var mu sync.Mutex

	nodes := make([]Node, 0)
	for _, node := range resolver.ResolverState.Addresses {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-b.stopChan:
				return
			default:
				b.mu.RLock()
				hcTimeout := b.cfg.HealthCheckTimeout
				b.mu.RUnlock()
				hc := HealthCheckerEx{timeout: hcTimeout}
				nodeInfo := Node{address: node.Addr, state: Closed, amountOfErrors: 0}

				state, err := hc.Check(node.Addr)
				if !state || err != nil {
					nodeInfo.state = Open
					nodeInfo.timer = *time.NewTicker(10 * time.Second)
				}
				mu.Lock()
				nodes = append(nodes, nodeInfo)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	b.mu.Lock()
	b.nodes = nodes
	b.mu.Unlock()

	return nil
}

// UpdateSubConnState вызывается gRPC при изменении состояния SubConn. Например, из Connecting в Transient_failure
func (b customBalancer) UpdateSubConnState(subConn balancer.SubConn, s balancer.SubConnState) {
	return
}

func (b customBalancer) Close() {
	close(b.stopChan)
	b.wg.Wait()
}

type HealthCheckerEx struct {
	timeout time.Duration
}

func (hc *HealthCheckerEx) Check(addr string) (bool, error) {
	conn, err := net.DialTimeout("tcp", addr, hc.timeout)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

/*

grpc.Builder - интерфейс, который используется для создания кастомных баласировщиков.
При помощи метода grpc.Dial создается grpc-соединение с указанием кастомного балансировщика.
Здесь grpc.Dial - это метод, который создает независимое grpc-соединение клиента с наором backend-ов.

balancer.SubConn — это интерфейс, представляющий одно логическое соединение к бэкенду (например, к конкретному IP:порт).
Это не физическое TCP-соединение, а абстракция, которая управляет подключением и его состоянием
(например, CONNECTING, READY, IDLE, TRANSIENT_FAILURE).

Балансировщик создаёт, подключает, обновляет или удаляет SubConn в зависимости
от изменений адресов (от resolver'а) или их состояния (например, на основе health-check'ов).

В интерфейсе balancer.SubConn есть несколько ключевых методов, которыми управляет балансировщик:

1) Connect(): Инициирует подключение к бэкенду, если SubConn находится в состоянии IDLE.
Вызывается, например, в ExitIdle или после успешного health-check'а.

2) Shutdown(): Закрывает SubConn и освобождает ресурсы. Используется, если бэкенд
становится нездоровым (например, после MaxFails неудачных health-check'ов).

3) UpdateAddresses(addrs []resolver.Address): Обновляет адреса, связанные с SubConn, если resolver
предоставил новые метаданные для того же бэкенда.

--------------------------------------------------------------------------------------------------------------------------------------------------------

Picker — это интерфейс balancer.Picker, определённый в пакете google.golang.org/grpc/balancer.
Его основная задача — выбирать подходящий SubConn для каждого gRPC-запроса на основе политики балансировки
(например, round-robin, weighted round-robin, least connections и т.д.).
Метод Pick вызывается для каждого клиентского запроса и возвращает balancer.PickResult или ошибку, если нет доступных соединений.

Интерфейс Picker-а выглядит следующим образом:

type Picker interface {

	Выбирает SubConn для запроса. PickInfo содержит контекст и имя метода gRPC,
	что позволяет реализовать специфическую логику выбора.

    Pick(info PickInfo) (PickResult, error)
}

type PickInfo struct {
    FullMethodName string
    Ctx            context.Context
}

type PickResult struct {
    SubConn SubConn
    Done    func(DoneInfo)
}

----------------------------------------------------------------------------------------------------------------------------------------------------------

Circuit breaker - специальный паттерн, который регулирует отправку запросов.

Этот паттерн основывается на трех состояниях: закрытое, открытое и полуоткрытое.

Закрытое состояние - состояние, в котором можно проходить к защищаемому сервису.
То есть это нормальное рабочее состояние. В этом состоянии отслеживается количество неудачных запросов.
Если число ошибок не превышает определенный попрог - circuit breaker продолжает оставаться в закрытом состоянии.
Обычно ведется учет по времени ответа и количеству неудачных запросов.

Открытое состояние - состояние, в котором блокируются любые попытки выполнить запрос к защищаемому сервису.
Это профилактическая мера, которая проводится, чтобы сервис могут перезагрузится или починится. Переход в открытое
состояние происходит, когда количество неудачных запросов превышает некоторый порог (этот попрог может быть определен
количеством ошибок, временем ответа, комбинацией обоих факторов). После перехода в открытое состояние circuit breaker
находится в этом состоянии некоторый период времени.

Полуоткрытое соединение - переходное состояние, в котором circuit breaker переходит в полуоткрытое состояние. В этом состоянии
circuit breaker может частично отправлять запросы сервису, чтобы протестировать его доступность и надежность. После истечения времени
ожидания в открытом состоянии, circuit breaker переходит в полуоткрытое состояние. В этом состоянии он позволяет ограниченному
количеству запросов пройти к сервису. Если эти запросы успешно обработаны и не вызывают ошибок, circuit breaker возвращается
в закрытое состояние, считая, что проблемы с сервисом устранены. Если в полуоткрытом состоянии снова обнаруживаются ошибки,
circuit breaker снова переходит в открытое состояние, причем время ожидания начинает отсчитываться заново.

Есть несколько подходов для определения необходимости активации Circuit Breaker

1) Порог ошибок (Error threshold) - переход в открытое состояние определяется,
когда количество ошибок превышено заданный попрог в определенном временном интервале.

2) Процентный попрог ошибок - фиксируется общее количество запросов, а также количество неудачных запросов.
Circuit breaker активируется, когда процент ошибок от общего числа превышает заданный уровень. При достижении
определенного процента неудачных запросов circuit breaker активируется. Но после истечения определенного количества
времени процент ошибочных запросов сбрасывается.

3) Оценка времени ответа. Активация circuit breaker происходит, если среднее время ответа превышает установленный порог.
Причем считается среднее время отклика сервиса за указанный промежуток.

4) Можно использовать гибридный метод: отслеживание времени отклика запроса и количество неудачно завершившихся запросов.

ResolveNow в балансировщике gRPC — это механизм для принудительного обновления списка серверов,
позволяющий быстрее реагировать на изменения в инфраструктуре и улучшающий отказоустойчивость.

В этом варианте балансировщик должен один раз в некоторый период запрашивать информацию о существующих
сервисах у некоторого стороннего сервиса, например, DNS, Service Discovery и так далее.
И при обнаружении новых узлов в принудительном формате добавлять новые хосты в список сервисов
для перенаправления запросов.

----------------------------------------------------------------------------------------------------------------------------------------------------

Метрики. Предполагаемый список метрик:

RED:

Rate - количество запросов.
Errors - количество ошибок.
Duration - время обработки одного запроса.

*/
