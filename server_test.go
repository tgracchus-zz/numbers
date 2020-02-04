package numbers_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"tgracchus/numbers"
	"time"
)

const connectionReadTimeout = time.Duration(1) * time.Second

func TestNewSingleConnectionListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, client := net.Pipe()
	expectedNumber := "098765432"

	sendData(t, client, expectedNumber)
	cnnListener := numbers.NewSingleConnectionListener(NewMockTcpController(t, expectedNumber+"\n", nil),
		connectionReadTimeout)
	cnnListener(ctx, &mockListener{connection: []net.Conn{server}})
}

func TestNewSingleConnectionListenerContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server, client := net.Pipe()
	expectedNumber := "098765432"
	sendData(t, client, expectedNumber)

	controller := NewMockTcpController(t, "", nil)
	cnnListener := numbers.NewSingleConnectionListener(controller, connectionReadTimeout)
	cnnListener(ctx, &mockListener{connection: []net.Conn{server}})
}

func TestNewSingleConnectionListenerControllerReturnsErrorAndJustLogIt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, client := net.Pipe()
	expectedNumber := "098765432"
	sendData(t, client, expectedNumber)

	controller := NewMockTcpController(t, expectedNumber+"\n", nil)
	cnnListener := numbers.NewSingleConnectionListener(controller, connectionReadTimeout)
	cnnListener(ctx, &mockListener{connection: []net.Conn{server}})
}

func TestNewMultipleConnectionListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedNumber := "098765432"
	server, client := net.Pipe()
	sendData(t, client, expectedNumber)
	server2, client2 := net.Pipe()
	sendData(t, client2, expectedNumber)

	controller := NewMockTcpController(t, expectedNumber+"\n", nil)
	cnnListener := numbers.NewSingleConnectionListener(controller, connectionReadTimeout)
	multipleCnnListener, err := numbers.NewMultipleConnectionListener(2, cnnListener)
	if err != nil {
		t.Fatal(err)
	}
	multipleCnnListener(ctx, &mockListener{connection: []net.Conn{server, server2}, i: 0})
}

type mockListener struct {
	connection []net.Conn
	i          int
	mux        sync.Mutex
}

func (m *mockListener) Accept() (net.Conn, error) {
	if m.i < len(m.connection) {
		return nil, errors.New("stop")
	}
	conn := m.connection[m.i]
	m.mux.Lock()
	defer m.mux.Unlock()
	m.i = m.i + 1
	return conn, nil
}

func (m *mockListener) Close() error {
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return nil
}
func NewMockTcpController(t *testing.T, expectedData string, customError error) numbers.TCPController {
	return func(ctx context.Context, c net.Conn) error {
		reader := bufio.NewReader(c)
		data, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}

		if customError != nil {
			return customError
		}
		if data != expectedData {
			t.Fatal(fmt.Errorf("expected %s but got %s", expectedData, data))
		}
		return nil
	}
}

func BenchmarkServer(*testing.B) {

}
