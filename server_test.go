package numbers_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"tgracchus/numbers"
	"time"
)

func TestNewSingleConnectionListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, client := net.Pipe()
	expectedNumber := "098765432"

	sendData(t, client, expectedNumber)
	cnnListener := numbers.NewSingleConnectionListener(newMockTcpController(t, cancel, expectedNumber+"\n"))
	cnnListener(ctx, &mockListener{connection: server})
}

func TestNewSingleConnectionListenerContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server, client := net.Pipe()
	expectedNumber := "098765432"
	sendData(t, client, expectedNumber)

	controller := newMockTcpController(t, cancel, "")
	cnnListener := numbers.NewSingleConnectionListener(controller)
	cnnListener(ctx, &mockListener{connection: server})
}

func TestNewSingleConnectionListenerControllerReturnsErrorAndJustLogIt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, client := net.Pipe()
	expectedNumber := "098765432"
	sendData(t, client, expectedNumber)

	controller := newMockTcpController(t, cancel, expectedNumber+"\n")
	cnnListener := numbers.NewSingleConnectionListener(controller)
	cnnListener(ctx, &mockListener{connection: server})
}

func TestNewMultipleConnectionListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockConnectionListener := newMockConnectionListener(t, ctx, cancel)
	multipleCnnListener, err := numbers.NewMultipleConnectionListener([]numbers.ConnectionListener{mockConnectionListener})
	if err != nil {
		t.Fatal(err)
	}

	server, _ := net.Pipe()
	multipleCnnListener(ctx, &mockListener{connection: server, i: 0})
}

func newMockConnectionListener(t *testing.T, ctx context.Context, cancel context.CancelFunc) numbers.ConnectionListener {
	ticker := time.NewTicker(time.Second)
	mock := func(ctx context.Context, l net.Listener) {
		cancel()
	}
	go func() {
		select {
		case <-ticker.C:
			cancel()
			t.Fatal("expected to be closed")
		case <-ctx.Done():
		}
	}()
	return mock
}

type mockListener struct {
	connection net.Conn
	i          int
	mux        sync.Mutex
}

func (m *mockListener) Accept() (net.Conn, error) {
	return m.connection, nil
}

func (m *mockListener) Close() error {
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return nil
}
func newMockTcpController(t *testing.T, cancel context.CancelFunc, expectedData string) numbers.TCPController {
	return func(ctx context.Context, c net.Conn) error {
		reader := bufio.NewReader(c)
		data, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}

		if data != expectedData {
			t.Fatal(fmt.Errorf("expected %s but got %s", expectedData, data))
		}
		cancel()
		return nil
	}
}
