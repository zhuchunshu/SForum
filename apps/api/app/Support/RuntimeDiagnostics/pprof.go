// Package runtimediagnostics owns opt-in, loopback-only process diagnostics.
package runtimediagnostics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"strings"
	"sync"
	"time"
)

const shutdownTimeout = 2 * time.Second

type Server struct {
	server    *http.Server
	listener  net.Listener
	closeOnce sync.Once
	closeErr  error
}

// StartPprof 启动独立于公开 Fiber 路由的诊断监听器。
// 即使显式启用，也只接受 loopback 地址，避免堆、goroutine 和命令行泄漏到公网。
func StartPprof(ctx context.Context, enabled bool, addr string, logger *slog.Logger) (*Server, error) {
	if !enabled {
		return nil, nil
	}
	addr = strings.TrimSpace(addr)
	if err := validateLoopbackAddress(addr); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen pprof: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", httppprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", httppprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", httppprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", httppprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", httppprof.Trace)
	for _, profile := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("/debug/pprof/"+profile, httppprof.Handler(profile))
	}

	server := &Server{
		server: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       30 * time.Second,
		},
		listener: listener,
	}
	if logger != nil {
		logger.Info("runtime pprof enabled", "addr", listener.Addr().String())
	}
	go func() {
		if serveErr := server.server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && logger != nil {
			logger.Error("runtime pprof stopped unexpectedly", "error", serveErr)
		}
	}()
	if ctx != nil && ctx.Done() != nil {
		done := ctx.Done()
		go func() {
			<-done
			_ = server.Close()
		}()
	}
	return server, nil
}

func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		s.closeErr = s.server.Shutdown(ctx)
	})
	return s.closeErr
}

func validateLoopbackAddress(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid pprof address: %w", err)
	}
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("pprof address must use a loopback host")
	}
	return nil
}
