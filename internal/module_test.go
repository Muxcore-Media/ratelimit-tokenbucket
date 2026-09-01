package internal

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
)

func TestModuleInfo(t *testing.T) {
	Version = "0.1.2"
	m := NewModule(Config{})
	info := m.Info()
	if info.ID == "" {
		t.Error("module ID must not be empty")
	}
	if info.Version != "0.1.2" {
		t.Errorf("version = %q want 0.1.2", info.Version)
	}
	if info.MinCoreVersion != "0.5.8" {
		t.Errorf("MinCoreVersion=%q want 0.5.8", info.MinCoreVersion)
	}
	if len(info.Contracts) == 0 || info.Contracts[0].Interface != "RateLimiterProvider" {
		t.Errorf("expected RateLimiterProvider contract, got %+v", info.Contracts)
	}
}

func TestModuleLifecycle(t *testing.T) {
	m := NewModule(Config{GRPCAddr: "127.0.0.1:0"})
	ctx := context.Background()

	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Health(ctx); err != nil {
		t.Fatalf("Health after Start: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := m.Health(ctx); err == nil {
		t.Fatal("Health after Stop should fail")
	}
}

func TestAllowDisabledAlwaysAllows(t *testing.T) {
	m := NewModule(Config{Enabled: false})
	ctx := context.Background()

	resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetAllowed() {
		t.Fatal("disabled limiter must allow empty key")
	}

	resp, err = m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "client-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetAllowed() {
		t.Fatal("disabled limiter must allow requests")
	}
}

func TestAllowEnabledRejectsEmptyKey(t *testing.T) {
	m := NewModule(Config{Enabled: true})
	ctx := context.Background()

	_, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: ""})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code=%v want InvalidArgument", status.Code(err))
	}
}

func TestAllowBurstExhaustionAndRefill(t *testing.T) {
	m := NewModule(Config{Enabled: true, Rate: 10, Burst: 2})
	ctx := context.Background()
	key := "burst-key"

	for i := 0; i < 2; i++ {
		resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: key})
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetAllowed() {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: key})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetAllowed() {
		t.Fatal("third request should be denied after burst exhaustion")
	}

	time.Sleep(150 * time.Millisecond)

	resp, err = m.Allow(ctx, &ratelimitv1.AllowRequest{Key: key})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetAllowed() {
		t.Fatal("expected refill after 1/rate seconds")
	}
}

func TestAllowPerKeyIsolation(t *testing.T) {
	m := NewModule(Config{Enabled: true, Rate: 1, Burst: 1})
	ctx := context.Background()

	resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "key-a"})
	if err != nil || !resp.GetAllowed() {
		t.Fatalf("key-a first: resp=%v err=%v", resp, err)
	}
	resp, err = m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "key-a"})
	if err != nil || resp.GetAllowed() {
		t.Fatalf("key-a second should deny: resp=%v err=%v", resp, err)
	}

	resp, err = m.Allow(ctx, &ratelimitv1.AllowRequest{Key: "key-b"})
	if err != nil || !resp.GetAllowed() {
		t.Fatalf("key-b should still be allowed: resp=%v err=%v", resp, err)
	}
}

func TestEnabledRPC(t *testing.T) {
	m := NewModule(Config{Enabled: true})
	ctx := context.Background()

	resp, err := m.Enabled(ctx, &ratelimitv1.EnabledRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetEnabled() {
		t.Fatal("expected enabled=true")
	}

	m.mu.Lock()
	m.enabled = false
	m.mu.Unlock()

	resp, err = m.Enabled(ctx, &ratelimitv1.EnabledRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetEnabled() {
		t.Fatal("expected enabled=false")
	}
}

func TestIdleBucketEviction(t *testing.T) {
	m := NewModule(Config{Enabled: true, Rate: 10, Burst: 5, IdleTTL: 50 * time.Millisecond})
	ctx := context.Background()
	key := "idle-key"

	resp, err := m.Allow(ctx, &ratelimitv1.AllowRequest{Key: key})
	if err != nil || !resp.GetAllowed() {
		t.Fatalf("initial allow: resp=%v err=%v", resp, err)
	}

	m.mu.Lock()
	if _, ok := m.buckets[key]; !ok {
		m.mu.Unlock()
		t.Fatal("bucket should exist after Allow")
	}
	m.mu.Unlock()

	time.Sleep(80 * time.Millisecond)

	m.evictIdleBuckets(time.Now())

	m.mu.Lock()
	_, ok := m.buckets[key]
	m.mu.Unlock()
	if ok {
		t.Fatal("stale bucket should be evicted")
	}

	resp, err = m.Allow(ctx, &ratelimitv1.AllowRequest{Key: key})
	if err != nil || !resp.GetAllowed() {
		t.Fatalf("allow after eviction should recreate bucket: resp=%v err=%v", resp, err)
	}
}

func TestNewModuleEnvOverride(t *testing.T) {
	t.Setenv("RATELIMIT_RATE", "50")
	t.Setenv("RATELIMIT_BURST", "75")
	t.Setenv("RATELIMIT_ENABLED", "true")
	t.Setenv("RATELIMIT_IDLE_TTL", "5m")

	m := NewModule(Config{})
	if m.rate != 50 {
		t.Errorf("rate=%v want 50", m.rate)
	}
	if m.burst != 75 {
		t.Errorf("burst=%v want 75", m.burst)
	}
	if !m.enabled {
		t.Error("expected enabled=true from env")
	}
	if m.idleTTL != 5*time.Minute {
		t.Errorf("idleTTL=%v want 5m", m.idleTTL)
	}
}

func TestNewModuleInvalidEnvKeepsDefaults(t *testing.T) {
	t.Setenv("RATELIMIT_RATE", "-1")
	t.Setenv("RATELIMIT_BURST", "0")
	t.Setenv("RATELIMIT_IDLE_TTL", "not-a-duration")

	m := NewModule(Config{})
	if m.rate != 100 {
		t.Errorf("rate=%v want default 100", m.rate)
	}
	if m.burst != 200 {
		t.Errorf("burst=%v want default 200", m.burst)
	}
	if m.idleTTL != defaultIdleTTL {
		t.Errorf("idleTTL=%v want default %v", m.idleTTL, defaultIdleTTL)
	}
}

func TestDefaultGRPCAddrLoopback(t *testing.T) {
	m := NewModule(Config{})
	if m.grpcAddr != "127.0.0.1:9800" {
		t.Errorf("grpcAddr=%q want 127.0.0.1:9800", m.grpcAddr)
	}
}
