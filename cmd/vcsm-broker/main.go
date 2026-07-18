package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/broker"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/ipc"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/platform"
)

func main() {
	paths, err := platform.DefaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vcsm-broker:", err)
		os.Exit(1)
	}
	vaultPath := flag.String("vault", paths.Vault, "encrypted vault database")
	endpoint := flag.String("endpoint", paths.Endpoint, "local IPC endpoint")
	flag.Parse()

	service, err := broker.New(*vaultPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vcsm-broker:", err)
		os.Exit(1)
	}
	defer service.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := ipc.Serve(ctx, *endpoint, service.Handle); err != nil {
		fmt.Fprintln(os.Stderr, "vcsm-broker:", err)
		os.Exit(1)
	}
}
