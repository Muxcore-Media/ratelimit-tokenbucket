package client_test

import (
	"context"
	"net"
	"testing"

	"github.com/Muxcore-Media/core/pkg/contracts"
	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
	"github.com/Muxcore-Media/ratelimit-tokenbucket/internal"
	"github.com/Muxcore-Media/ratelimit-tokenbucket/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1 << 20

func startTestModule(t *testing.T, cfg internal.Config) *bufconn.Listener {
	t.Helper()
	mod := internal.NewModule(cfg)
	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer()
	ratelimitv1.RegisterRateLimitServiceServer(gs, mod)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})
	return lis
}

func dialClient(t *testing.T, lis *bufconn.Listener) *client.Client {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return client.New(conn)
}

func TestClientAllowWhenEnabled(t *testing.T) {
	lis := startTestModule(t, internal.Config{Enabled: true, Rate: 10, Burst: 5})
	rl := dialClient(t, lis)

	if !rl.Allow(context.Background(), "client-ip") {
		t.Fatal("expected allow")
	}
	if !rl.Enabled() {
		t.Fatal("expected enabled")
	}
}

func TestClientAllowWhenDisabled(t *testing.T) {
	lis := startTestModule(t, internal.Config{Enabled: false})
	rl := dialClient(t, lis)

	if !rl.Allow(context.Background(), "client-ip") {
		t.Fatal("disabled sidecar must allow")
	}
	if rl.Enabled() {
		t.Fatal("expected disabled")
	}
}

func TestClientAllowFailOpenOnRPCError(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer()
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})

	rl := dialClient(t, lis)
	if !rl.Allow(context.Background(), "client-ip") {
		t.Fatal("RPC errors must fail open (allow)")
	}
	if rl.Enabled() {
		t.Fatal("RPC errors must report disabled")
	}
}

var _ contracts.RateLimiterProvider = (*client.Client)(nil)
