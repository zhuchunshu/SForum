package extensionsruntime_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type protocolTestSettings map[string]string

func (s protocolTestSettings) ListSettings(context.Context, string) (map[string]string, error) {
	return s, nil
}

func startSMTPProbeServer(t *testing.T) (string, int, func(*testing.T)) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for SMTP probe: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		if _, err := io.WriteString(connection, "220 localhost ESMTP ready\r\n"); err != nil {
			done <- err
			return
		}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- err
				return
			}
			command := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(command, "EHLO "), strings.HasPrefix(command, "HELO "):
				_, err = io.WriteString(connection, "250-localhost\r\n250 OK\r\n")
			case command == "QUIT":
				_, err = io.WriteString(connection, "221 bye\r\n")
				done <- err
				return
			default:
				err = fmt.Errorf("unexpected SMTP command %q", command)
			}
			if err != nil {
				done <- err
				return
			}
		}
	}()
	address := listener.Addr().(*net.TCPAddr)
	wait := func(t *testing.T) {
		t.Helper()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("SMTP probe server: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("SMTP probe server did not finish")
		}
	}
	return address.IP.String(), address.Port, wait
}

func protocolTestRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve protocol fixture path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../.."))
}

func temporaryPluginWorkspace(t *testing.T, repositoryRoot, pluginModuleRoot string) string {
	t.Helper()
	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	if goVersion == runtime.Version() || strings.ContainsAny(goVersion, " \t\r\n") {
		t.Fatalf("unsupported Go runtime version %q", runtime.Version())
	}
	body := fmt.Sprintf("go %s\n\nuse (\n\t%s\n\t%s\n)\n",
		goVersion,
		strconv.Quote(filepath.ToSlash(filepath.Join(repositoryRoot, "apps/api"))),
		strconv.Quote(filepath.ToSlash(pluginModuleRoot)),
	)
	path := filepath.Join(t.TempDir(), "go.work")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temporary plugin workspace: %v", err)
	}
	return path
}
