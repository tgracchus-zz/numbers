package numbers

import (
	"context"
	"github.com/pkg/errors"
	"log"
	"net"
	"os"
)

const numberLogFileName = "numbers.log"

// ConnectionListener given a listener it listen and establish connections.
type ConnectionListener func(ctx context.Context, l net.Listener)

type TCPController func(ctx context.Context, c net.Conn, numbers chan int, terminate chan int) error

// StartNumberServer start the number server tcp application with
// number of concurrent server connections and at the given address.
func StartNumberServer(concurrentConnections int, address string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if concurrentConnections < 0 {
		log.Panicf("concurrency level should be more than 0, not %d", concurrentConnections)
	}
	conf := &net.ListenConfig{KeepAlive: 15}
	l, err := conf.Listen(ctx, "tcp", address)
	if err != nil {
		log.Printf("%v", errors.Wrap(err, "Star listener"))
		return
	}
	defer closeListener(l)
	log.Printf("server started at:%s", address)

	connections := NewSingleConnectionListener(l)

	terminate := make(chan int)
	numbersOuts := make([]chan int, concurrentConnections)
	for i := 0; i < concurrentConnections; i++ {
		numbers := DefaultTCPController(connections, terminate)
		numbersOuts[i] = numbers
	}

	deDuplicatedNumbers := NumberStore(reportPeriod, numbersOuts, terminate)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	done := FileWriter(deDuplicatedNumbers, dir+"/"+numberLogFileName)

	cancelContextWhenTerminateSignal(cancel, terminate, done)

	<-done
}

func cancelContextWhenTerminateSignal(cancel context.CancelFunc,
	terminate chan int, done chan int) chan int {
	go func() {
		for {
			select {
			case <-terminate:
				cancel()
			case <-done:
				return
			}
		}
	}()
	return terminate
}

func closeListener(l net.Listener) {
	log.Printf("%v", "closing listener")
	if err := l.Close(); err != nil {
		log.Printf("%v", errors.Wrap(err, "Closing listener"))
	}
}

// NewSingleConnectionListener creates a new ConnectionListener which listen for a connection
// and then it calls the given TCPController in a sync way.
func NewSingleConnectionListener(l net.Listener) chan net.Conn {
	connections := make(chan net.Conn)
	go func() {
		defer close(connections)
		for {
			c, err := l.Accept()
			if err != nil {
				log.Printf("%v", errors.Wrap(err, "accept connection"))
				return
			}
			connections <- c
		}
	}()
	return connections
}

func closeConnection(c net.Conn) {
	if c != nil {
		if err := c.Close(); err != nil {
			log.Printf("%v", errors.Wrap(err, "closeConnection"))
		}
	}
}
