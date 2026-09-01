package test

import (
	"context"
	"net"
	"testing"

	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
	"github.com/Muxcore-Media/ratelimit-tokenbucket/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1 << 20

func startGRPC(t *testing.T, cfg internal.Config) (ratelimitv1.RateLimitServiceClient, func()) {
	t.Helper()
	mod := internal.NewModule(cfg)
	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer()
	ratelimitv1.RegisterRateLimitServiceServer(gs, mod)
	go func() { _ = gs.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		_ = conn.Close()
		gs.Stop()
		_ = lis.Close()
	}
	return ratelimitv1.NewRateLimitServiceClient(conn), cleanup
}

func TestAllowEnabledGRPC(t *testing.T) {
	client, cleanup := startGRPC(t, internal.Config{Enabled: true, Rate: 10, Burst: 2})
	defer cleanup()
	ctx := context.Background()

	resp, err := client.Allow(ctx, &ratelimitv1.AllowRequest{Key: "10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetAllowed() {
		t.Fatal("expected allowed")
	}
}

func TestEnabledGRPC(t *testing.T) {
	client, cleanup := startGRPC(t, internal.Config{Enabled: true})
	defer cleanup()
	ctx := context.Background()

	resp, err := client.Enabled(ctx, &ratelimitv1.EnabledRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetEnabled() {
		t.Fatal("expected enabled")
	}
}
