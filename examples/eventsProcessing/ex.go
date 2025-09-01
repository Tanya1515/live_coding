package main

import (
	"fmt"
	"time"
)

/*
События поступают последовательно, но не непрерывно — могут быть пропуски, дубли, задержки.
Они доступны через итератор (например, из Kafka, очереди или файла).

Цель — агрегировать события по пользователям и сохранять промежуточное состояние на диск, чтобы при перезапуске продолжить с последнего обработанного события.
*/

// Event — событие от пользователя
type Event struct {
	UserID    uint32
	Action    string
	Timestamp int64 // Unix-время
}

// EventSource — источник событий
type EventSource interface {
	// ReadNext возвращает следующее событие
	// Второе значение — true, если событие получено
	// Реализация сама обрабатывает сбои и переподключения
	ReadNext() (*Event, bool)
}

// StateStore — хранилище состояния
type StateStore interface {
	// Load возвращает последний Timestamp и агрегаты
	// Если нет данных — возвращает 0 и пустую мапу
	Load() (int64, map[uint32]map[string]int, error)

	// Save сохраняет состояние
	Save(int64, map[uint32]map[string]int) error
}

const (
	batchSize = 100
)

// ProcessEvents обрабатывает события и сохраняет агрегаты
// Если firstRun == true — начинает с начала
// Если firstRun == false — продолжает с последнего состояния
func ProcessEvents(source EventSource, stateStore StateStore, firstRun bool) error {
	var maxTime int64
	timeLast, userEvents, err := stateStore.Load()
	if err != nil {
		return fmt.Errorf("error %v: error while getting data from state service %v", time.Now(), err)
	}
	event, ok := source.ReadNext()
	if !firstRun {
		for (event.Timestamp != timeLast) || ok {
			event, ok = source.ReadNext()
		}
		if !ok {
			return nil
		}
	}
	if timeLast > event.Timestamp {
		maxTime = timeLast
	} else {
		maxTime = event.Timestamp
	}
	events := make([]*Event, batchSize)
	for {
		if len(events) == batchSize {
			// вынести в функцию
			for _, event := range events {
				if _, exists := userEvents[event.UserID]; !exists {
					userEvents[event.UserID] = make(map[string]int, 10)
				}
				userEvents[event.UserID][event.Action] += 1
				if maxTime < event.Timestamp {
					maxTime = event.Timestamp
				}

			}
			err := stateStore.Save(maxTime, userEvents)
			if err != nil {
				return fmt.Errorf("error %v: error while storing data to state service %s", time.Now(), err)
			}
			events = events[:0]
		}

		event, ok = source.ReadNext()
		events = append(events, event)
		if !ok {
			break
		}
	}
	// отправить batch
	return nil
}
