package numbers

import (
	"context"
	"github.com/pkg/errors"
	"log"
	"net"
)

// ConnectionListener given a listener it listen and establish connections.
type ConnectionListener interface {
	Listen(ctx context.Context, l net.Listener)
	Close()
}

// TCPController is executed when a ConnectionListener establish a connection,
// it defines the way to handle a connection.
type TCPController interface {
	Handle(ctx context.Context, c net.Conn) error
	Close()
}

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
	go connectionListener.Listen(ctx, l)
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
	return &multipleListener{listeners: listeners}
}

type multipleListener struct {
	listeners [] ConnectionListener
}

func (ml *multipleListener) Listen(ctx context.Context, l net.Listener) {
	for i := 0; i < len(ml.listeners); i++ {
		go func(index int) {
			log.Printf("creating connection handler: %d", index)
			ml.listeners[index].Listen(ctx, l)
		}(i)
	}
}

func (ml *multipleListener) Close() {
	for _, listener := range ml.listeners {
		listener.Close()
	}
}

// NewSingleConnectionListener creates a new ConnectionListener which listen for a connection
// and then it calls the given TCPController in a sync way.
func NewSingleConnectionListener(controller TCPController) ConnectionListener {
	return &defaultListener{controller: controller}
}

type defaultListener struct {
	controller TCPController
}

func (dl *defaultListener) Listen(ctx context.Context, l net.Listener) {
	for {
		if err := listenOnce(ctx, l, dl.controller); err != nil {
			log.Printf("%v", err)
			return
		}
	}
}

func (dl *defaultListener) Close() {
	dl.controller.Close()
}

func listenOnce(ctx context.Context, l net.Listener, controller TCPController) error {
	c, err := l.Accept()
	if err != nil {
		return errors.Wrap(err, "accept connection")
	}
	defer closeConnection(c)
	err = controller.Handle(ctx, c)
	if err != nil {
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
