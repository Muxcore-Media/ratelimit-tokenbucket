package internal

import (
	"context"
	"testing"
)

func TestModuleInfo(t *testing.T) {
	m := NewModule(Config{})
	info := m.Info()
	if info.ID == "" {
		t.Error("module ID must not be empty")
	}
	if info.Version == "" {
		t.Error("module version must not be empty")
	}
	if info.HTTPAddr != "127.0.0.1:9800" {
		t.Errorf("default gRPC addr = %q", info.HTTPAddr)
	}
}

func TestResolveGRPCAddr(t *testing.T) {
	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "true")
	t.Setenv("MUXCORE_GRPC_INSECURE", "")
	if got := resolveGRPCAddr(":9800"); got != "127.0.0.1:9800" {
		t.Fatalf("insecure wildcard = %q", got)
	}
	if got := resolveGRPCAddr("0.0.0.0:9800"); got != "127.0.0.1:9800" {
		t.Fatalf("insecure 0.0.0.0 = %q", got)
	}

	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "")
	if got := resolveGRPCAddr(":9800"); got != ":9800" {
		t.Fatalf("secure wildcard = %q", got)
	}
}

func TestModuleLifecycle(t *testing.T) {
	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "true")
	m := NewModule(Config{GRPCAddr: "127.0.0.1:0"})
	ctx := context.Background()

	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
