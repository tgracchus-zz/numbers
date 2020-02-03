package main

import (
	"context"
	"log"
	"numbers"
)

func main() {
	numbersProtocol, terminate := numbers.NewNumbersProtocol(10)
	cnnHandler := numbers.NewDefaultConnectionHandler(numbersProtocol)
	concurrentHandler, err := numbers.NewConcurrentConnectionHandler(5, cnnHandler)
	if err != nil {
		log.Println(err)
	}
	numbers.Start(context.Background(), concurrentHandler, terminate)
}
