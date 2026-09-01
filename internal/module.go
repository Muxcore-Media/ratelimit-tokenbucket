package internal

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Muxcore-Media/core/pkg/contracts"
	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
)

const defaultIdleTTL = 10 * time.Minute

type bucket struct {
	tokens     float64
	lastTick   time.Time
	lastAccess time.Time
}

type Module struct {
	ratelimitv1.UnimplementedRateLimitServiceServer
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64
	burst    int
	enabled  bool
	idleTTL  time.Duration
	grpcSrv  *grpc.Server
	lis      net.Listener
	id       string
	grpcAddr string
}

type Config struct {
	ID       string
	GRPCAddr string
	Rate     float64
	Burst    int
	Enabled  bool
	IdleTTL  time.Duration
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "ratelimit-tokenbucket"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = "127.0.0.1:9800"
	}
	if cfg.Rate <= 0 {
		cfg.Rate = 100
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 200
	}
	if cfg.IdleTTL == 0 {
		cfg.IdleTTL = defaultIdleTTL
	}
	if v := os.Getenv("RATELIMIT_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = strings.TrimSpace(v)
	}
	if v := os.Getenv("RATELIMIT_RATE"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil || f <= 0 {
			slog.Warn("ignoring invalid RATELIMIT_RATE", "value", v, "error", err)
		} else {
			cfg.Rate = f
		}
	}
	if v := os.Getenv("RATELIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err != nil || n <= 0 {
			slog.Warn("ignoring invalid RATELIMIT_BURST", "value", v, "error", err)
		} else {
			cfg.Burst = n
		}
	}
	if v := os.Getenv("RATELIMIT_ENABLED"); v != "" {
		cfg.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("RATELIMIT_IDLE_TTL"); v != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err != nil || d <= 0 {
			slog.Warn("ignoring invalid RATELIMIT_IDLE_TTL", "value", v, "error", err)
		} else {
			cfg.IdleTTL = d
		}
	}
	return &Module{
		id:       cfg.ID,
		grpcAddr: cfg.GRPCAddr,
		rate:     cfg.Rate,
		burst:    cfg.Burst,
		enabled:  cfg.Enabled,
		idleTTL:  cfg.IdleTTL,
		buckets:  make(map[string]*bucket),
	}
}

func (m *Module) Info() contracts.ModuleInfo {
	ver := Version
	if ver == "" {
		ver = "0.0.0-dev"
	}
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Rate Limit Token Bucket",
		Version:      ver,
		Roles:        []string{"infrastructure"},
		Description:  "Per-key token bucket rate limiter (process-local buckets; disabled until RATELIMIT_ENABLED)",
		Author:       "MuxCore",
		Capabilities: []string{contracts.CapabilityRateLimiter, "ratelimit.tokenbucket", "settings"},
		Contracts: []contracts.ContractDeclaration{
			{
				Repo:      "github.com/Muxcore-Media/core/pkg/contracts",
				Interface: "RateLimiterProvider",
				Version:   "v0.5.8",
			},
		},
		MinCoreVersion: "0.5.8",
		HTTPAddr:       m.grpcAddr,
	}
}

func (m *Module) Init(ctx context.Context) error {
	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis
	slog.Info("ratelimit initialized", "rate", m.rate, "burst", m.burst, "enabled", m.enabled, "idle_ttl", m.idleTTL, "addr", m.grpcAddr)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	srv := grpc.NewServer()
	lis := m.lis
	m.grpcSrv = srv
	ratelimitv1.RegisterRateLimitServiceServer(srv, m)
	modulesdk.RegisterSettings(srv, m.id, m)
	go func() {
		slog.Info("ratelimit gRPC service started", "addr", m.grpcAddr)
		if err := srv.Serve(lis); err != nil {
			slog.Error("ratelimit gRPC serve error", "error", err)
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
		m.grpcSrv = nil
	}
	if m.lis != nil {
		_ = m.lis.Close()
		m.lis = nil
	}
	slog.Info("ratelimit stopped")
	return nil
}

func (m *Module) Health(ctx context.Context) error {
	if m.lis == nil || m.grpcSrv == nil {
		return fmt.Errorf("gRPC server not serving")
	}
	return nil
}

func (m *Module) Allow(ctx context.Context, req *ratelimitv1.AllowRequest) (*ratelimitv1.AllowResponse, error) {
	m.mu.Lock()
	enabled := m.enabled
	m.mu.Unlock()
	if !enabled {
		return &ratelimitv1.AllowResponse{Allowed: true}, nil
	}
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}
	allowed := m.allow(req.GetKey())
	return &ratelimitv1.AllowResponse{Allowed: allowed}, nil
}

func (m *Module) Enabled(ctx context.Context, req *ratelimitv1.EnabledRequest) (*ratelimitv1.EnabledResponse, error) {
	m.mu.Lock()
	enabled := m.enabled
	m.mu.Unlock()
	return &ratelimitv1.EnabledResponse{Enabled: enabled}, nil
}

func (m *Module) evictIdleBuckets(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, b := range m.buckets {
		if !b.lastAccess.IsZero() && now.Sub(b.lastAccess) >= m.idleTTL {
			delete(m.buckets, key)
		}
	}
}

func (m *Module) allow(key string) bool {
	now := time.Now()
	m.evictIdleBuckets(now)

	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.buckets[key]
	if !ok {
		m.buckets[key] = &bucket{
			tokens:     float64(m.burst) - 1,
			lastTick:   now,
			lastAccess: now,
		}
		return true
	}

	b.lastAccess = now
	elapsed := now.Sub(b.lastTick).Seconds()
	b.lastTick = now

	b.tokens = math.Min(b.tokens+elapsed*m.rate, float64(m.burst))
	if b.tokens < 1 {
		slog.Info("ratelimit deny", "key", key, "remaining_tokens", b.tokens)
		return false
	}
	b.tokens--
	return true
}
