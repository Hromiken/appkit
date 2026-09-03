package httpfx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewServerAppliesDefaults(t *testing.T) {
	server, err := NewServer(Config{}, http.NewServeMux())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if server.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", server.Addr)
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s, want 5s", server.ReadHeaderTimeout)
	}
}

func TestConfigRejectsNegativeTimeout(t *testing.T) {
	if err := (Config{ReadTimeout: -time.Second}).Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative timeout error")
	}
}

func TestServerLifecycleStartsAndStopsListener(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	listener := newBlockingListener()
	server, err := NewServer(Config{}, http.NewServeMux())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	registerLifecycle(lifecycleParams{
		Lifecycle: lifecycle,
		Shutdown:  shutdownerFunc(func(...fx.ShutdownOption) error { return nil }),
		Config:    Config{ShutdownTimeout: time.Second},
		Server:    server,
		Logger:    zap.NewNop(),
		Listen: func(string, string) (net.Listener, error) {
			return listener, nil
		},
	})

	lifecycle.RequireStart()
	lifecycle.RequireStop()
	select {
	case <-listener.closed:
	case <-time.After(time.Second):
		t.Fatal("listener was not closed")
	}
}

func TestServerLifecycleReturnsListenError(t *testing.T) {
	wantErr := errors.New("listen failed")
	lifecycle := fxtest.NewLifecycle(t)
	server, err := NewServer(Config{}, http.NewServeMux())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	registerLifecycle(lifecycleParams{
		Lifecycle: lifecycle,
		Shutdown:  shutdownerFunc(func(...fx.ShutdownOption) error { return nil }),
		Config:    Config{},
		Server:    server,
		Logger:    zap.NewNop(),
		Listen: func(string, string) (net.Listener, error) {
			return nil, wantErr
		},
	})

	if err := lifecycle.Start(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Lifecycle.Start() error = %v, want %v", err, wantErr)
	}
}

func TestUnexpectedServeErrorRequestsShutdown(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	serveErr := errors.New("accept failed")
	listener := &errorListener{err: serveErr}
	shutdownCalled := make(chan struct{}, 1)
	server, err := NewServer(Config{}, http.NewServeMux())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	registerLifecycle(lifecycleParams{
		Lifecycle: lifecycle,
		Shutdown: shutdownerFunc(func(...fx.ShutdownOption) error {
			shutdownCalled <- struct{}{}
			return nil
		}),
		Config: Config{},
		Server: server,
		Logger: zap.NewNop(),
		Listen: func(string, string) (net.Listener, error) {
			return listener, nil
		},
	})

	lifecycle.RequireStart()
	select {
	case <-shutdownCalled:
	case <-time.After(time.Second):
		t.Fatal("unexpected Serve error did not request shutdown")
	}
	lifecycle.RequireStop()
}

type shutdownerFunc func(...fx.ShutdownOption) error

func (function shutdownerFunc) Shutdown(options ...fx.ShutdownOption) error {
	return function(options...)
}

type testAddr string

func (address testAddr) Network() string { return "test" }
func (address testAddr) String() string  { return string(address) }

type blockingListener struct {
	closed    chan struct{}
	closeOnce sync.Once
}

func newBlockingListener() *blockingListener {
	return &blockingListener{closed: make(chan struct{})}
}

func (listener *blockingListener) Accept() (net.Conn, error) {
	<-listener.closed
	return nil, net.ErrClosed
}

func (listener *blockingListener) Close() error {
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}

func (*blockingListener) Addr() net.Addr { return testAddr("blocking") }

type errorListener struct {
	err error
}

func (listener *errorListener) Accept() (net.Conn, error) { return nil, listener.err }
func (*errorListener) Close() error                       { return nil }
func (*errorListener) Addr() net.Addr                     { return testAddr("error") }
