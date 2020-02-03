package main

import (
	"context"
	"log"
	"tgracchus/numbers"
)

func main() {
	numbersProtocol, terminate := numbers.NewNumbersProtocol()
	cnnHandler := numbers.NewDefaultConnectionHandler(numbersProtocol, 5)
	concurrentHandler, err := numbers.NewConcurrentConnectionHandler(5, cnnHandler)
	if err != nil {
		log.Println(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-terminate:
				cancel()
				return
			}
		}
	}()

	numbers.Start(ctx, concurrentHandler)
}
