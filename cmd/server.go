package main

import (
	"golang.org/x/sync/semaphore"
	"numbers"
)

func main() {
	numbersPipe := make(chan int)
	defer close(numbersPipe)
	numbersProtocol := numbers.NewNumbersProtocol(numbersPipe)

	sem := semaphore.NewWeighted(int64(1))
	eventLoop := numbers.NewLimitedEventLoop(sem, numbersProtocol)

	numbers.Start(eventLoop)
}
