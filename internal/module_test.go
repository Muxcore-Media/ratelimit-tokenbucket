package internal

import (
	"context"
	"testing"

	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
)

func testConfig() Config {
	return Config{Rate: 100, Burst: 10, Enabled: true, GRPCAddr: ":0"}
}

func TestModuleInfo(t *testing.T) {
	m := NewModule(Config{})
	info := m.Info()
	if info.ID == "" {
		t.Error("module ID must not be empty")
	}
	if info.Version == "" {
		t.Error("module version must not be empty")
	}
	if info.MinCoreVersion == "" {
		t.Error("MinCoreVersion must not be empty")
	}
	if len(info.Contracts) == 0 {
		t.Error("Contracts must not be empty")
	}
}

func TestModuleLifecycle(t *testing.T) {
	m := NewModule(testConfig())
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

func TestAllow(t *testing.T) {
	m := NewModule(testConfig())
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer m.Stop(ctx)

	// First request should be allowed (burst=10)
	resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "user:1"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Allowed {
		t.Fatal("expected first request to be allowed")
	}
}

func TestAllowExhaustBurst(t *testing.T) {
	m := NewModule(Config{Rate: 1000, Burst: 3, Enabled: true, GRPCAddr: ":0"})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer m.Stop(ctx)

	for i := 0; i < 3; i++ {
		resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "user:1"})
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Allowed {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestAllowDisabled(t *testing.T) {
	m := NewModule(Config{Rate: 100, Burst: 10, Enabled: false, GRPCAddr: ":0"})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer m.Stop(ctx)

	resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "user:1"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Allowed {
		t.Fatal("expected allowed when disabled")
	}
}

func TestAllowEmptyKey(t *testing.T) {
	m := NewModule(testConfig())
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer m.Stop(ctx)

	_, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: ""})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestEnabled(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		m := NewModule(Config{Rate: 100, Burst: 10, Enabled: enabled, GRPCAddr: ":0"})
		ctx := context.Background()
		if err := m.Init(ctx); err != nil {
			t.Fatalf("Init: %v", err)
		}
		defer m.Stop(ctx)

		resp, err := m.Enabled(ctx, &ratelimitv1.EnabledRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Enabled != enabled {
			t.Fatalf("expected enabled=%v, got %v", enabled, resp.Enabled)
		}
	}
}

func TestPerKeyBuckets(t *testing.T) {
	m := NewModule(Config{Rate: 1000, Burst: 2, Enabled: true, GRPCAddr: ":0"})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer m.Stop(ctx)

	resp, _ := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "user:a"})
	if !resp.Allowed {
		t.Fatal("user:a first request allowed")
	}

	resp, _ = m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "user:b"})
	if !resp.Allowed {
		t.Fatal("user:b first request allowed")
	}
}
