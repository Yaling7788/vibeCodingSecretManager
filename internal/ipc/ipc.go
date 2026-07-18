package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/api"
)

const maxFrame = 1 << 20

type Handler func(context.Context, api.Request, func(api.Response) error)

func Serve(ctx context.Context, endpoint string, handler Handler) error {
	listener, err := listen(endpoint)
	if err != nil {
		return err
	}
	defer listener.Close()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go serveConn(ctx, conn, handler)
	}
}

func serveConn(ctx context.Context, conn net.Conn, handler Handler) {
	defer conn.Close()
	decoder := json.NewDecoder(io.LimitReader(conn, maxFrame))
	var request api.Request
	if err := decoder.Decode(&request); err != nil {
		return
	}
	encoder := json.NewEncoder(conn)
	handler(ctx, request, func(response api.Response) error {
		response.ID = request.ID
		return encoder.Encode(response)
	})
}

func Call(ctx context.Context, endpoint string, request api.Request, onResponse func(api.Response) error) error {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("connect to broker: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return err
	}
	decoder := json.NewDecoder(bufio.NewReader(conn))
	for {
		var response api.Response
		if err := decoder.Decode(&response); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := onResponse(response); err != nil {
			return err
		}
		if response.Final {
			return nil
		}
	}
}

func Wait(ctx context.Context, endpoint string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		conn, err := dial(ctx, endpoint)
		if err == nil {
			conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
