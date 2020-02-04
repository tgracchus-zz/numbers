package main

import (
	"context"
	"log"
	"tgracchus/numbers"
)

func main() {
	numbersProtocol, numbersChn, terminate := numbers.NewNumbersProtocol(10)

	cnnHandler := numbers.NewDefaultConnectionHandler(numbersProtocol, 5)
	concurrentHandler, err := numbers.NewConcurrentConnectionHandler(5, cnnHandler)
	if err != nil {
		log.Println(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := numbers.NewNumberStore(10, numbersChn)
	numbers.StartStore(ctx, store)

	go func() {
		for {
			select {
			case <-terminate:
				cancel()
				return
			}
		}
	}()

	numbers.StartServer(ctx, concurrentHandler)
}
