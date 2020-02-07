package numbers_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"tgracchus/numbers"
	"time"
)

func TestNewSingleConnectionListener(t *testing.T) {
	server, client := net.Pipe()
	expectedNumber := "098765432"
	sendData(t, client, expectedNumber)

	connections := numbers.NewSingleConnectionListener(&mockListener{connection: server})
	ticker := time.NewTicker(time.Second)
	select {
	case connection := <-connections:
		if connection != server {
			t.Fatal("expected server connection here")
		}
	case <-ticker.C:
		t.Fatal("expected connection here")
	}
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
