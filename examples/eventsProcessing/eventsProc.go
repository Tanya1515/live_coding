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
	bufferSize = 100
	amountOfAttempts = 2
)

func sendEvents(eventChan chan []Event, result chan error, storeData map[uint32]map[string]int, stateStore StateStore, maxTime int64) {
	defer close(result)
	attempt := 0
	for events := range eventChan {
		for _, event := range events {
			if event.Timestamp > maxTime {
				maxTime = event.Timestamp
			}
			if _, exists := storeData[event.UserID]; !exists {
				storeData[event.UserID] = make(map[string]int, 10)
			}
			storeData[event.UserID][event.Action] += 1
		}

		err := stateStore.Save(maxTime, storeData)
		for err != nil && attempt < amountOfAttempts {
			attempt++
			time.Sleep(time.Duration(attempt)*time.Second)
			err = stateStore.Save(maxTime, storeData)
		}
		if err != nil {
			result <- fmt.Errorf("error %v: error while saving data to state store %v", time.Now(), err)
			return
		}
	}
	return
}

// ProcessEvents обрабатывает события и сохраняет агрегаты
// Если firstRun == true — начинает с начала
// Если firstRun == false — продолжает с последнего состояния

func ProcessEvents(source EventSource, stateStore StateStore, firstRun bool) error {
	event, nextExist := source.ReadNext()
	if !nextExist {
		return nil
	}
	lastTime, storedData, err := stateStore.Load()
	if err != nil {
		return fmt.Errorf("error %v: error while getting stored data %v", time.Now(), err)
	}
	if !firstRun {
		for nextExist && event.Timestamp < lastTime {
			event, nextExist = source.ReadNext()
		}

		if event.Timestamp > lastTime {
			lastTime = event.Timestamp
		}

		if !nextExist {
			return nil
		}
	}

	events := make([]Event, 0, bufferSize)
	chanEvents := make(chan []Event)
	chanError := make(chan error)

	go sendEvents(chanEvents, chanError, storedData, stateStore, lastTime)

	for {
		select {
		case err := <-chanError:
			close(chanEvents)
			return err
		default:
			if len(events) == bufferSize {
				chanEvents <- events
				events = events[:0]
			}

			event, nextExist = source.ReadNext()
			if !nextExist {
				if len(events) != 0 {
					chanEvents <- events
				}
				close(chanEvents)
				err := <-chanError
				return err
			}
			events = append(events, *event)
		}
	}
}
