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
)


func sendEvents(eventChan chan []*Event, errChan chan error, stateStore StateStore, mapUsers map[uint32]map[string]int, lastStamp int64) {
    defer close(errChan)
    
    for events := range eventChan {
        for _, event := range events {
            if _, exists := mapUsers[event.UserID]; !exists {
                mapUsers[event.UserID] = make(map[string]int, 0)
            }
            mapUsers[event.UserID][event.Action] += 1
            if lastStamp < event.Timestamp {
                lastStamp = event.Timestamp
            }
        }
        err := stateStore.Save(lastStamp, mapUsers)
        if err != nil {
           errChan <- fmt.Errorf("error at %v while saving data to state store: %w", time.Now(), err)
           return
        }
    }
    
    return
}

func ProcessEvents(source EventSource, stateStore StateStore, firstRun bool) error {

    lastStamp, mapUsers, err := stateStore.Load()
    if err != nil {
        return fmt.Errorf("error at %v while loading data from store: %w", time.Now(), err)
    }
    var eventToProc *Event
    var exists bool
    if !firstRun {
        for {
            eventToProc, exists = source.ReadNext()
            if !exists {
                return nil 
            }
            if lastStamp < eventToProc.Timestamp {
                lastStamp = eventToProc.Timestamp
                break
            }
        }
    }
    
    eventBuf := make([]*Event, 0, bufferSize)
    eventBuf = append(eventBuf, eventToProc)
    eventChan := make(chan []*Event, 1)
    errChan := make(chan error, 1)

    go sendEvents(eventChan, errChan, stateStore, mapUsers, lastStamp)

    for {
        select {
            case err := <- errChan:
                close(eventChan)
                return err
            default: 
                eventToProc, exists = source.ReadNext()
                if exists {
                    if len(eventBuf) >= bufferSize {
                        eventChan <- eventBuf
                        eventBuf = eventBuf[:0]
                    }
                    eventBuf = append(eventBuf, eventToProc)
                    continue
                }
                
                if len(eventBuf) > 0 {
                    eventChan <- eventBuf
                }
                
                close(eventChan)
                
                return <- errChan
        }
    }
}




