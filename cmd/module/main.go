package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Muxcore-Media/ratelimit-tokenbucket/internal"
)

var version = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		os.Exit(runHealthCheck())
	}
	internal.Version = version
	mod := internal.NewModule(internal.Config{})
	insecureTLS := os.Getenv("MUXCORE_INSECURE_DISABLE_TLS") == "true" || os.Getenv("MUXCORE_GRPC_INSECURE") == "true"
	if err := modulesdk.Run(modulesdk.Config{
		Module:   mod,
		Insecure: insecureTLS,
	}); err != nil {
		slog.Error("module exited", "error", err)
		os.Exit(1)
	}
}

func runHealthCheck() int {
	addr := os.Getenv("RATELIMIT_GRPC_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9800"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			host, port = "127.0.0.1", strings.TrimPrefix(addr, ":")
		} else {
			fmt.Fprintf(os.Stderr, "health-check: bad RATELIMIT_GRPC_ADDR: %v\n", err)
			return 1
		}
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	dialAddr := net.JoinHostPort(host, port)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(dialAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health-check: dial: %v\n", err)
		return 1
	}
	defer func() { _ = conn.Close() }()

	client := ratelimitv1.NewRateLimitServiceClient(conn)
	if _, err := client.Enabled(ctx, &ratelimitv1.EnabledRequest{}); err != nil {
		fmt.Fprintf(os.Stderr, "health-check: Enabled RPC: %v\n", err)
		return 1
	}
	return 0
}
