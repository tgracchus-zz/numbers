package numbers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

func Start(ctx context.Context, connectionHandler ConnectionHandler, terminate chan bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Println(err)
		return
	}
	defer closeListener(l)

	go func() {
		for {
			select {
			case <-terminate:
				cancel()
				log.Println("closing listener")
				closeListener(l)
				return
			}
		}
	}()

	connectionHandler(ctx, l)
}

func closeListener(l net.Listener) {
	if err := l.Close(); err != nil {
		log.Println(err)
	}
}

type ConnectionHandler func(ctx context.Context, l net.Listener)

func NewConcurrentConnectionHandler(concurrencyLevel int, cnnHandler ConnectionHandler) (ConnectionHandler, error) {
	if concurrencyLevel < 0 {
		return nil, fmt.Errorf("concurrency level should be more than 0, not %d", concurrencyLevel)
	}
	return func(ctx context.Context, l net.Listener) {
		var wg sync.WaitGroup
		wg.Add(concurrencyLevel)
		for i := 0; i < concurrencyLevel; i++ {
			go func(connection int) {
				log.Printf("creating connection handler: %d", connection)
				cnnHandler(ctx, l)
				log.Printf("closing connection handler: %d", connection)
				wg.Done()
			}(i)
		}
		wg.Wait()
	}, nil
}

func NewDefaultConnectionHandler(protocol TCPProtocol) ConnectionHandler {
	return func(ctx context.Context, l net.Listener) {
		for {
			if checkIfTerminated(ctx) {
				return
			}
			if err := connect(ctx, l, protocol); err != nil {
				log.Print(err)
				return
			}
		}
	}
}

func connect(ctx context.Context, l net.Listener, protocol TCPProtocol) error {
	c, err := l.Accept()
	if err != nil {
		return err
	}
	defer closeConnection(c)

	if checkIfTerminated(ctx) {
		return nil
	}

	log.Printf("connection: %s, accepted", c.RemoteAddr().String())
	if err := protocol(ctx, c); err != nil {
		log.Println(err)
	}
	return nil
}

func closeConnection(c net.Conn) {
	if err := c.Close(); err != nil {
		log.Println(err)
	}
}

type TCPProtocol func(ctx context.Context, c net.Conn) error

func checkIfTerminated(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
