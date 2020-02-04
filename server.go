package numbers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type ConnectionHandler func(ctx context.Context, l net.Listener)

type TCPProtocol func(ctx context.Context, c net.Conn) error

func StartServer(ctx context.Context, connectionHandler ConnectionHandler) {
	l, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Println(err)
		return
	}
	defer closeListener(l)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
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

func NewDefaultConnectionHandler(protocol TCPProtocol, readTimeOut int) ConnectionHandler {
	return func(ctx context.Context, l net.Listener) {
		for {
			if checkIfTerminated(ctx) {
				return
			}
			if err := connectAndInvoke(ctx, l, protocol, readTimeOut); err != nil {
				log.Print(err)
				return
			}
		}
	}
}

func connectAndInvoke(ctx context.Context, l net.Listener, protocol TCPProtocol, readTimeOut int) error {
	c, err := l.Accept()
	if err != nil {
		return err
	}
	defer closeConnection(c)
	log.Printf("connection: %s, accepted", c.RemoteAddr().String())

	if err := c.SetReadDeadline(time.Now().Add(time.Duration(readTimeOut) * time.Second)); err != nil {
		return fmt.Errorf("connection: %s, SetReadDeadline failed: %s", c.RemoteAddr().String(), err)
	}
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

func checkIfTerminated(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
