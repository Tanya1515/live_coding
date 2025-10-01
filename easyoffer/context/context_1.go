package main

import (
	"context"
	"fmt"
	"time"
)

/*

1) Создается контекст с таймаутом на 3 секунды

2) Затем программа засыпает на 2950 Millisecond, остается 50 Millisecond

3) Затем запускается doDbRequest 

3.1) Создается дочерний контекст с таймером на 10 секунд. 

3.2) Создается таймер на 1 секунду. 

3.3) Выполняется select и он будет заблокирован, пока не выполнится одно из условий. 
Если бы был default, то select сразу бы выполнился и doDbRequest сразу завершился. 

3.4) В силу того, что у родительского контекста закончится таймаут через 50 Millisecond, 
он отработает быстрее и будет напечатано Timeout. 

*/

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	time.Sleep(2950 * time.Millisecond)
	doDbRequest(ctx)
}

func doDbRequest(ctx context.Context) {
	newCtx, _ := context.WithTimeout(ctx, 10*time.Second)
	timer := time.NewTimer(1 * time.Second)
	select {
	case <-newCtx.Done():
		fmt.Println("Timeout")
	case <-timer.C:
		fmt.Println("Request Done")
	}
}
