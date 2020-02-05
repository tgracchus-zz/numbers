package numbers

import (
	"context"
	"github.com/pkg/errors"
	"log"
	"net"
)

type ConnectionListener func(ctx context.Context, l net.Listener)

type TCPController func(ctx context.Context, c net.Conn) error

func StartServer(ctx context.Context, connectionListener ConnectionListener, address string) {
	conf := &net.ListenConfig{KeepAlive: 15}
	l, err := conf.Listen(ctx, "tcp", address)
	if err != nil {
		log.Printf("%v", errors.Wrap(err, "StartConnectionListener"))
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
				log.Printf("%v", "closing listener")
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

func NewMultipleConnectionListener(cnnHandlers [] ConnectionListener) (ConnectionListener, error) {
	return func(ctx context.Context, l net.Listener) {
		for i := 0; i < len(cnnHandlers); i++ {
			log.Printf("creating connection handler: %d", i)
			go cnnHandlers[i](ctx, l)
		}
		<-ctx.Done()
	}, nil
}

func NewSingleConnectionListener(controller TCPController) ConnectionListener {
	return func(ctx context.Context, l net.Listener) {
		for {
			if listenOnce(ctx, l, controller) {
				return
			}
		}
	}
}

func listenOnce(ctx context.Context, l net.Listener, controller TCPController) bool {
	if checkIfTerminated(ctx) {
		return true
	}
	c, err := l.Accept()
	if err != nil {
		log.Printf("%v", errors.Wrap(err, "accept connection"))
		return true
	}
	defer closeConnection(c)

	if checkIfTerminated(ctx) {
		return true
	}
	err = controller(ctx, c)
	if err != nil {
		log.Printf("%v", errors.Wrap(err, "controller error"))
	}
	return false
}

func closeConnection(c net.Conn) {
	if c != nil {
		if err := c.Close(); err != nil {
			log.Printf("%v", errors.Wrap(err, "closeConnection"))
		}
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
