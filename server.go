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

func Start(ctx context.Context, l net.Listener, connectionHandler ConnectionHandler) {
	connectionHandler(ctx, l)
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
