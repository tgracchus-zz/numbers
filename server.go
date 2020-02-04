package numbers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type ConnectionListener func(ctx context.Context, l net.Listener)

type TCPController func(ctx context.Context, c net.Conn) error

func StartServer(ctx context.Context, connectionListener ConnectionListener, address string) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	defer closeListener(l)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//Cancel
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

	connectionListener(ctx, l)
}

func closeListener(l net.Listener) {
	if err := l.Close(); err != nil {
		log.Println(err)
	}
}

func NewMultipleConnectionListener(concurrencyLevel int, cnnHandler ConnectionListener) (ConnectionListener, error) {
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

func NewSingleConnectionListener(protocol TCPController, readTimeOut int) ConnectionListener {
	return func(ctx context.Context, l net.Listener) {
		for {
			if checkIfTerminated(ctx) {
				return
			}
			if err := listenAndInvoke(ctx, l, protocol, readTimeOut); err != nil {
				log.Print(err)
				return
			}
		}
	}
}

func listenAndInvoke(ctx context.Context, l net.Listener, protocol TCPController, readTimeOut int) error {
	c, err := l.Accept()
	if err != nil {
		return err
	}
	defer closeConnection(c)
	log.Printf("connection: %s, accepted", c.RemoteAddr().String())

	if err := c.SetReadDeadline(time.Now().Add(time.Duration(readTimeOut) * time.Second)); err != nil {
		return fmt.Errorf("connection: %s, SetReadDeadline failed: %s", c.RemoteAddr().String(), err)
	}
	return protocol(ctx, c)
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
