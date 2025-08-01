package main

// Задача: Создайте конвейер (pipeline), где каждый этап обрабатывает данные и
// может возвращать ошибку. Реализуйте функцию RunPipeline(stages ...Stage),
// где Stage — это функция вида func(in <-chan int) (out <-chan int, err error).

// Пример:
// stage1 := func(in <-chan int) (<-chan int, error) {
//     out := make(chan int)
//     go func() {
//         defer close(out)
//         for num := range in {
//             out <- num * 2
//         }
//     }()

//     return out, nil
// }

type Stage func(in <-chan int) (<-chan int, error)

func RunPipeline(stages ...Stage) ([]int, error) {
	// Реализуйте
	return nil, nil
}
