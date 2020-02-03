package main

import (
	"context"
	"log"
	"net"
	"tgracchus/numbers"
)

func main() {
	numbersProtocol, terminate := numbers.NewNumbersProtocol()
	cnnHandler := numbers.NewDefaultConnectionHandler(numbersProtocol, 5)
	concurrentHandler, err := numbers.NewConcurrentConnectionHandler(5, cnnHandler)
	if err != nil {
		log.Println(err)
	}
	l, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Println(err)
		return
	}
	defer closeListener(l)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-terminate:
				cancel()
				log.Println("closing listener")
				_ = l.Close()
				return
			}
		}
	}()

	numbers.Start(ctx, l, concurrentHandler)
}

func closeListener(l net.Listener) {
	if err := l.Close(); err != nil {
		log.Println(err)
	}
}