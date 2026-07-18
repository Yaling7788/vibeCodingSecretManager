//go:build !windows

package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func listen(endpoint string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(endpoint); err == nil {
		conn, dialErr := net.Dial("unix", endpoint)
		if dialErr == nil {
			conn.Close()
			return nil, fmt.Errorf("broker is already listening at %s", endpoint)
		}
		if err := os.Remove(endpoint); err != nil {
			return nil, err
		}
	}
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(endpoint, 0o600); err != nil {
		listener.Close()
		return nil, err
	}
	return cleanupListener{Listener: listener, path: endpoint}, nil
}

type cleanupListener struct {
	net.Listener
	path string
}

func (l cleanupListener) Close() error {
	err := l.Listener.Close()
	_ = os.Remove(l.path)
	return err
}

func dial(ctx context.Context, endpoint string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", endpoint)
}
