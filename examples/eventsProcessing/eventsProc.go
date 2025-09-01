package main

// /*
// События поступают последовательно, но не непрерывно — могут быть пропуски, дубли, задержки.
// Они доступны через итератор (например, из Kafka, очереди или файла).

// Цель — агрегировать события по пользователям и сохранять промежуточное состояние на диск, чтобы при перезапуске продолжить с последнего обработанного события.
// */


// // Event — событие от пользователя
// type Event struct {
//     UserID    uint32
//     Action    string
//     Timestamp int64 // Unix-время
// }

// // EventSource — источник событий
// type EventSource interface {
//     // ReadNext возвращает следующее событие
//     // Второе значение — true, если событие получено
//     // Реализация сама обрабатывает сбои и переподключения
//     ReadNext() (*Event, bool)
// }

// // StateStore — хранилище состояния
// type StateStore interface {
//     // Load возвращает последний Timestamp и агрегаты
//     // Если нет данных — возвращает 0 и пустую мапу
//     Load() (int64, map[uint32]map[string]int, error)

//     // Save сохраняет состояние
//     Save(int64, map[uint32]map[string]int) error
// }

// // ProcessEvents обрабатывает события и сохраняет агрегаты
// // Если firstRun == true — начинает с начала
// // Если firstRun == false — продолжает с последнего состояния
// func ProcessEvents(source EventSource, stateStore StateStore, firstRun bool) error {
// 	return nil
// }
