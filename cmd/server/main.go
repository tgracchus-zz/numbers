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

	numbersController, numbersOut, terminateOut := numbers.NewNumbersController(10)
	deDuplicatedNumbers := numbers.NewNumberStore(ctx, 10, numbersOut, 10)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	numbers.NewFileWriter(ctx, deDuplicatedNumbers, dir+"/numbers.log")

	cnnListener := numbers.NewSingleConnectionListener(numbersController, 5)
	multipleListener, err := numbers.NewMultipleConnectionListener(5, cnnListener)
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

	numbers.StartServer(ctx, multipleListener, "localhost:1234")
}
