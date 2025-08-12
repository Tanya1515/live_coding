package main

import "time"

/*

Полное условие: Создайте обработчик сообщений Kafka с гарантией однократной обработки.

Требования:

 - Транзакционное сохранение данных и offset'ов
 - Параллельная обработка разных партиций
 - Retry при временных ошибках
 - Ручной commit offset'ов

Тесты:

 - Повторная обработка сообщений
 - Потеря сообщений при crash
 - Конкурентная обработка

*/

type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
}

type Storage interface {
	// Сохраняет offset атомарно с данными
	Save(tx interface{}, partition int32, offset int64, data []byte) error

	// Возвращает последний обработанный offset
	LastOffset(partition int32) (int64, error)
}

type Handler interface {
	Process(msg Message) (interface{}, error)
}

type ProcessorConfig struct {
	Workers       int
	CommitTimeout time.Duration
	MaxBatchSize  int
}

type ExactlyOnceProcessor struct {
	// ...
}
