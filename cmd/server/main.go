package main

import (
	"context"
	"log"
	"os"
	"tgracchus/numbers"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersProtocol, numbersIn, terminateOut := numbers.NewNumbersProtocol(10)
	deDuplicatedNumbers := numbers.NewNumberStore(ctx, 10, numbersIn, 10)

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = numbers.NewFileWriter(ctx, deDuplicatedNumbers, dir)
	if err != nil {
		log.Fatal(err)
	}

	cnnHandler := numbers.NewDefaultConnectionHandler(numbersProtocol, 5)
	concurrentHandler, err := numbers.NewConcurrentConnectionHandler(5, cnnHandler)
	if err != nil {
		log.Println(err)
	}

	go func() {
		for {
			select {
			case <-terminateOut:
				cancel()
				return
			}
		}
	}()

	numbers.StartServer(ctx, concurrentHandler)
}
