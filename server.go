package numbers

import (
	"golang.org/x/sync/semaphore"
	"log"
	"net"
)

func Start(eventLoop EventLoop) {
	l, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Println(err)
		return
	}
	defer closeListener(l)
	if err := eventLoop(l); err != nil {
		log.Println(err)
	}
}

func closeListener(l net.Listener) {
	if err := l.Close(); err != nil {
		log.Println(err)
	}
}

type EventLoop func(l net.Listener) error

func NewLimitedEventLoop(sem *semaphore.Weighted, protocol TCPProtocol) EventLoop {
	return func(l net.Listener) error {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Println(err)
				return err
			}
			client := c.RemoteAddr().String()
			//https://godoc.org/golang.org/x/sync/semaphore
			if !sem.TryAcquire(1) {
				log.Printf("client: %s, too many concurrent clients", client)
				CloseConnection(c)
				continue
			}

			go func() {
				//defer executing https://tour.golang.org/flowcontrol/13
				defer CloseConnection(c)
				defer sem.Release(1)
				log.Printf("client: %s, serving", client)
				if err := protocol(c); err != nil {
					log.Println(err)
				}
			}()
		}
	}
}

func CloseConnection(c net.Conn) {
	if err := c.Close(); err != nil {
		log.Println(err)
	}
}

type TCPProtocol func(c net.Conn) error
