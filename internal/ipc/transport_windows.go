//go:build windows

package ipc

import (
	"context"
	"net"

	"github.com/Microsoft/go-winio"
)

func listen(endpoint string) (net.Listener, error) {
	return winio.ListenPipe(endpoint, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;OW)",
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	})
}

func dial(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}
