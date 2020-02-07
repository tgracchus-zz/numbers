package numbers

import (
	"context"
	"github.com/pkg/errors"
	"log"
	"net"
)

// ConnectionListener given a listener it listen and establish connections.
type ConnectionListener func(ctx context.Context, l net.Listener)

type TCPController func(ctx context.Context, c net.Conn, numbers chan int, terminate chan int) error

// StartServer starts the server with the given connection listener and at the given address.
func StartServer(ctx context.Context, connectionListener ConnectionListener, address string) {
	conf := &net.ListenConfig{KeepAlive: 15}
	l, err := conf.Listen(ctx, "tcp", address)
	if err != nil {
		log.Printf("%v", errors.Wrap(err, "Star listener"))
		return
	}
	defer closeListener(l)
	log.Printf("server started at:%s", address)
	go connectionListener(ctx, l)
	<-ctx.Done()
	return
}

func closeListener(l net.Listener) {
	log.Printf("%v", "closing listener")
	if err := l.Close(); err != nil {
		log.Printf("%v", errors.Wrap(err, "Closing listener"))
	}
}

// NewMultipleConnectionListener starts has many instances as given in separate goroutines and waits the
// context to be cancelled.
func NewMultipleConnectionListener(listeners [] ConnectionListener) ConnectionListener {
	return func(ctx context.Context, l net.Listener) {
		for i := 0; i < len(listeners); i++ {
			go func(index int) {
				log.Printf("creating connection handler: %d", index)
				listeners[index](ctx, l)
			}(i)
		}
	}
}

// NewSingleConnectionListener creates a new ConnectionListener which listen for a connection
// and then it calls the given TCPController in a sync way.
func NewSingleConnectionListener(controller TCPController, terminate chan int) (ConnectionListener, chan int) {
	numbers := make(chan int)
	return func(ctx context.Context, l net.Listener) {
		defer close(numbers)
		for {
			if err := listenOnce(ctx, l, controller, numbers, terminate); err != nil {
				log.Printf("%v", err)
				return
			}
		}
	}, numbers
}

type terminateError struct {
}

func (e *terminateError) Error() string { return "TERMINATED" }

func listenOnce(ctx context.Context, l net.Listener, controller TCPController, numbers chan int, terminate chan int) error {
	c, err := l.Accept()
	if err != nil {
		return errors.Wrap(err, "accept connection")
	}
	defer closeConnection(c)
	err = controller(ctx, c, numbers, terminate)
	if err != nil {
		serr, ok := err.(*terminateError)
		if ok {
			return serr
		}
		log.Printf("%v", errors.Wrap(err, "controller error"))
		return nil
	}
	return nil
}

func closeConnection(c net.Conn) {
	if c != nil {
		if err := c.Close(); err != nil {
			log.Printf("%v", errors.Wrap(err, "closeConnection"))
		}
	}
}
