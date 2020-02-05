package numbers_test

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
)

func testServer(t *testing.T, concurrentConnections int, clientsNumber int, reqs int, port string) {
	var wg sync.WaitGroup
	wg.Add(clientsNumber)
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//go numbers.StartNumberServer(ctx, concurrentConnections, "localhost:"+port)
	clients(&wg, clientsNumber, reqs, port)
	wg.Wait()
	//sendTerminate(port)
	//cancel()
}

func TestServer_5connections_1clients_10000reqs(t *testing.T) {
	testServer(t, 5, 1, 1000, "1234")
}

func TestServer_5connections_5clients_1000reqs(t *testing.T) {
	testServer(t, 5, 5, 1000, "1234")
}

func TestServer_5connections_10clients_1000reqs(t *testing.T) {
	testServer(t, 5, 10, 1000, "1234")
}

func TestServer_5connections_5clients_10000reqs(t *testing.T) {
	testServer(t, 5, 5, 10000, "1234")
}

func TestServer_5connections_10clients_10000reqs(t *testing.T) {
	testServer(t, 5, 10, 10000, "1234")
}

func TestServer_5connections_50clients_10000reqs(t *testing.T) {
	testServer(t, 50, 50, 10000, "1234")
}

func TestServe_50connections_100clients_10000reqs(t *testing.T) {
	testServer(t, 50, 100, 10000, "1234")
}

func clients(wg *sync.WaitGroup, totalClients int, reqs int, port string) {
	for clientNumber := 0; clientNumber < totalClients; clientNumber++ {
		go client(wg, clientNumber, reqs, port)
	}
}

func sendTerminate(port string) {
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		log.Printf("Client connection error: %s", err)
	}
	defer conn.Close()
	send(conn, "terminate\n", 0)
}

func client(wg *sync.WaitGroup, clientNumber int, reqs int, port string) {
	log.Printf("Client #: %d", clientNumber)
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		log.Printf("Client %d connection error: %s", clientNumber, err)
		return
	}
	defer conn.Close()
	for i := 0; i < reqs; i++ {
		// send to socket
		number := fmt.Sprintf("%09d\n", rand.Intn(1000000000))
		send(conn, number, i)
	}
	wg.Done()
}

func send(conn net.Conn, msg string, clientNumber int) {
	_, err := fmt.Fprintf(conn, msg)
	if err != nil {
		log.Printf("Client %d with error: %s", clientNumber, err)
	}

}
