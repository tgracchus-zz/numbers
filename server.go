package numbers

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"net"
)

type ConnectionListener func(ctx context.Context, l net.Listener)

type TCPController func(ctx context.Context, c net.Conn) error

func StartServer(ctx context.Context, connectionListener ConnectionListener, address string) {
	conf := &net.ListenConfig{KeepAlive: 1000}
	l, err := conf.Listen(ctx, "tcp", address)
	if err != nil {
		log.Printf("%+v", errors.Wrap(err, "StartConnectionListener"))
		return
	}
	defer closeListener(l)
	log.Printf("server started at:%s", address)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//Cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				cancel()
				log.Printf("%+v", "closing listener")
				closeListener(l)
				return
			}
		}
	}()

	connectionListener(ctx, l)
}

func closeListener(l net.Listener) {
	if err := l.Close(); err != nil {
		log.Printf("%v", errors.Wrap(err, "Closing listener"))
	}
}

func NewMultipleConnectionListener(concurrencyLevel int, cnnHandler ConnectionListener) (ConnectionListener, error) {
	if concurrencyLevel < 0 {
		return nil, fmt.Errorf("concurrency level should be more than 0, not %d", concurrencyLevel)
	}
	return func(ctx context.Context, l net.Listener) {
		for i := 0; i < concurrencyLevel; i++ {
			listener := func(connection int) {
				log.Printf("creating connection handler: %d", connection)
				cnnHandler(ctx, l)
				log.Printf("closing connection handler: %d", connection)
			}
			go listener(i)
		}
		<-ctx.Done()
	}, nil
}

func NewSingleConnectionListener(controller TCPController) ConnectionListener {
	return func(ctx context.Context, l net.Listener) {
		for {
			if checkIfTerminated(ctx) {
				return
			}
			c, err := l.Accept()
			if err != nil {
				log.Printf("%+v", errors.Wrap(err, "accept connection"))
			}
			if checkIfTerminated(ctx) {
				return
			}

			err = controller(ctx, c)
			if err != nil {
				log.Printf("%+v", errors.Wrap(err, "controller error"))
			}
		}
	}
}

func closeConnection(c net.Conn) {
	if err := c.Close(); err != nil {
		log.Printf("%+v", errors.Wrap(err, "closeConnection"))
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
