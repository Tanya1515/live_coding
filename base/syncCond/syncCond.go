package main

import (
	"math/rand"
	"sync"
	"time"
)

// sync.Cond - тип в языке программирования в Golang, который предоставляет
// условные переменные для синхронизации горутин в ситуациях,
// требующих условного выполнения.
//
//
// var cond = sync.NewCond(&mu) - создается условная переменная cond,
// которая управляет координацией с помощью мьютекса mu. Переменная
// типа sync.Cond содержит локер-поле L типа sync.Locker, значениями
// которого выступают *sync.Mutex и *sync.RWMutex.

// Основные методы:

// 1) Wait - разблокирует локер L и вводит текущую горутину в режим
// ожидания до получения сигнала. При получении сигнала локер L блокируется,
// и выполняется следующий после Wait код.

// 2) Signal - будит одну горутину, ожидающую условную переменную. Если ни
// одна горутина не ожидает, вызов этого метода не будет иметь эффекта.

// 3) Broadcast - разблокирует все горутины в очереди.

// Доступны два варианта:

// (*Cond) Signal() — разблокирует одну из ожидающих горутин, если такие есть.

/*

cond.L.Lock()           // Шаг 1: Захватываем мьютекс
for !condition() {      // Шаг 2: Проверяем условие (под защитой мьютекса)
    cond.Wait()         // Шаг 3: ВНУТРИ Wait() происходит:
                        //   - Шаг 3.1: Отпускаем мьютекс
                        //   - Шаг 3.2: Входим в ожидание
                        //   - Шаг 3.3: Получаем сигнал
                        //   - Шаг 3.4: Захватываем мьютекс
                        //   - Шаг 3.5: Возвращаемся из Wait()
    // Шаг 4: Wait() завершился, продолжаем цикл
    // (код здесь НЕ выполняется, если условие не изменилось)
}
// Шаг 5: Условие выполнилось, выполняем код
// выполняется некоторый код
cond.L.Unlock()         // Шаг 6: Освобождаем мьютекс

*/

// Причем другя горутина должна вызвать либо Signal, либо Broadcast, чтобы разблокировать горутину.

var pokemonList = []string{"Pikachu", "Charmander", "Squirtle", "Bulbasaur", "Jigglypuff"}
var cond = sync.NewCond(&sync.Mutex{})
var pokemon = ""

func main() {
	// Consumer
	go func() {
		cond.L.Lock()
		defer cond.L.Unlock()

		// waits until Pikachu appears
		for pokemon != "Pikachu" {
			cond.Wait()
		}
		println("Caught" + pokemon)
		pokemon = ""
	}()

	// Producer
	go func() {
		// Every 1ms, a random Pokémon appears
		for i := 0; i < 100; i++ {
			time.Sleep(time.Millisecond)

			cond.L.Lock()
			pokemon = pokemonList[rand.Intn(len(pokemonList))]
			cond.L.Unlock()

			cond.Signal()
		}
	}()

	time.Sleep(100 * time.Millisecond) // lazy wait
}
