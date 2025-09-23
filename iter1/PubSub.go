package main

// Задача:
// Реализуйте систему PubSub, где подписчики могут получать сообщения по темам.

// Интерфейс:

// type PubSub interface {
// 	Subscribe(topic string) <-chan interface{}
// 	Unsubscribe(topic string, ch <-chan interface{})
// 	Publish(topic string, msg interface{})
// }

import (
	"sync"
)

type PubSub struct {
	mapTopicsToSubcribes map[string]map[chan interface{}]struct{}
	mu                   *sync.RWMutex
	closed               bool
}

func NewPubSub() *PubSub {
	mapPubSub := make(map[string]map[chan interface{}]struct{}, 10)

	return &PubSub{
		mu:                   &sync.RWMutex{},
		mapTopicsToSubcribes: mapPubSub,
		closed:               false,
	}
}

func (ps *PubSub) Subscribe(topic string) <-chan interface{} {
	chanToWrite := make(chan interface{}, 1)
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		close(chanToWrite)
		return chanToWrite
	}
	if _, exists := ps.mapTopicsToSubcribes[topic]; !exists {
		ps.mapTopicsToSubcribes[topic] = make(map[chan interface{}]struct{}, 10)
	}
	ps.mapTopicsToSubcribes[topic][chanToWrite] = struct{}{}

	return chanToWrite
}

func (ps *PubSub) Unsubscribe(topic string, ch <-chan interface{}) {

	chanToClose := make(chan interface{}, 1)

	ps.mu.RLock()
	if ps.closed {
		ps.mu.RUnlock()
		close(chanToClose)
		return
	}
	for channel := range ps.mapTopicsToSubcribes[topic] {
		if channel == ch {
			chanToClose = channel
		}
	}
	ps.mu.RUnlock()

	ps.mu.Lock()
	if _, exists := ps.mapTopicsToSubcribes[topic][chanToClose]; exists {
		delete(ps.mapTopicsToSubcribes[topic], chanToClose)
		close(chanToClose)
	}
	ps.mu.Unlock()

}

func (ps *PubSub) Publish(topic string, msg interface{}) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if ps.closed {
		return
	}
	for channel := range ps.mapTopicsToSubcribes[topic] {
		select {
		case channel <- msg:
		default:
		}
	}

}

func (ps *PubSub) Close() {
	var wg sync.WaitGroup

	ps.mu.Lock()
	ps.closed = true
	for topic := range ps.mapTopicsToSubcribes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for channel := range ps.mapTopicsToSubcribes[topic] {
				delete(ps.mapTopicsToSubcribes[topic], channel)
				close(channel)
			}
		}()
	}
	wg.Wait()
	ps.mu.Unlock()
}
