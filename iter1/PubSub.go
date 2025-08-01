package main

// Задача:
// Реализуйте систему PubSub, где подписчики могут получать сообщения по темам.

// Интерфейс:

// type PubSub interface {
// 	Subscribe(topic string) <-chan interface{}
// 	Unsubscribe(topic string, ch <-chan interface{})
// 	Publish(topic string, msg interface{})
// }

import _ "sync"

type PubSub struct {
	// Добавьте поля
}

func NewPubSub() *PubSub {
	return &PubSub{}
}

func (ps *PubSub) Subscribe(topic string) <-chan interface{} {
	//
	return nil
}

func (ps *PubSub) Unsubscribe(topic string, ch <-chan interface{}) {
	//
}

func (ps *PubSub) Publish(topic string, msg interface{}) {
	//
}
