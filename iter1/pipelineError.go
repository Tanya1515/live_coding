package main

import "fmt"

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
	inputChan := make(chan int)
	result := make([]int, 0)

	go func() {
		defer close(inputChan)
		for i := range 10 {
			inputChan <- i
		}
	}()

	resultChanOut, err := stages[0](inputChan)
	if err != nil {
		fmt.Println("Error on stage: ", err)
		return nil, err
	}

	for key, stage := range stages {
		if key != 0 {
			resultChan, err := stage(resultChanOut)
			if err != nil {
				fmt.Println("Error on stage: ", err)
				return nil, err
			}
			resultChanOut = resultChan
		}
	}

	for value := range resultChanOut {
		result = append(result, value)
	}

	return result, nil
}

func main() {
	Stage1 := func(in <-chan int) (<-chan int, error) {
		out := make(chan int)
		go func() {
			defer close(out)
			for num := range in {
				out <- num * 2
			}
		}()

		return out, nil
	}

	Stage2 := func(in <-chan int) (<-chan int, error) {
		out := make(chan int)
		go func() {
			defer close(out)
			for num := range in {
				out <- num + 2
			}
		}()

		return out, nil
	}

	Stage3 := func(in <-chan int) (<-chan int, error) {
		out := make(chan int)
		go func() {
			defer close(out)
			for num := range in {
				out <- num * 10
			}
		}()

		return out, nil
	}

	result, err := RunPipeline(Stage1, Stage2, Stage3)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, value := range result {
		fmt.Println(value)
	}
}
