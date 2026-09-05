package internal

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/Muxcore-Media/core/pkg/contracts"
	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"github.com/Muxcore-Media/ratelimit-tokenbucket/internal/grpctls"
)

type bucket struct {
	tokens   float64
	lastTick time.Time
}

type Module struct {
	ratelimitv1.UnimplementedRateLimitServiceServer
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64
	burst    int
	enabled  bool
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
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "ratelimit-tokenbucket"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = "127.0.0.1:9800"
	}
	if v := os.Getenv("RATELIMIT_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	cfg.GRPCAddr = resolveGRPCAddr(cfg.GRPCAddr)
	if cfg.Rate <= 0 {
		cfg.Rate = 100
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 200
	}
	if v := os.Getenv("RATELIMIT_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Rate = f
		}
	}
	if v := os.Getenv("RATELIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Burst = n
		}
	}
	if v := os.Getenv("RATELIMIT_ENABLED"); v != "" {
		cfg.Enabled = v == "true" || v == "1"
	}
	return &Module{
		id:       cfg.ID,
		grpcAddr: cfg.GRPCAddr,
		rate:     cfg.Rate,
		burst:    cfg.Burst,
		enabled:  cfg.Enabled,
		buckets:  make(map[string]*bucket),
	}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Rate Limit Token Bucket",
		Version:      "0.1.3",
		Roles:        []string{"infrastructure"},
		Description:  "Per-key token bucket rate limiter",
		Author:       "MuxCore",
		Capabilities: []string{contracts.CapabilityRateLimiter, "ratelimit.tokenbucket", "settings"},
		HTTPAddr:     m.grpcAddr,
	}
}

func (m *Module) Init(ctx context.Context) error {
	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis
	slog.Info("ratelimit initialized", "rate", m.rate, "burst", m.burst, "enabled", m.enabled, "addr", m.grpcAddr)
	return nil
}

func resolveGRPCAddr(addr string) string {
	if !grpctls.InsecureAllowed() {
		return addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		return "127.0.0.1:" + port
	}
	return addr
}

func (m *Module) Start(ctx context.Context) error {
	tlsCfg, err := grpctls.ServerConfig()
	if err != nil {
		return fmt.Errorf("gRPC TLS: %w", err)
	}
	var grpcOpts []grpc.ServerOption
	if tlsCfg != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	} else {
		slog.Warn("ratelimit gRPC listening without TLS (dev only)",
			"addr", m.grpcAddr,
			"hint", "set MUXCORE_INSECURE_DISABLE_TLS only for local development")
	}
	m.grpcSrv = grpc.NewServer(grpcOpts...)
	ratelimitv1.RegisterRateLimitServiceServer(m.grpcSrv, m)
	modulesdk.RegisterSettings(m.grpcSrv, m.id, m)
	go func() {
		slog.Info("ratelimit gRPC service started", "addr", m.grpcAddr)
		if err := m.grpcSrv.Serve(m.lis); err != nil {
			slog.Error("ratelimit gRPC serve error", "error", err)
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
	}
	slog.Info("ratelimit stopped")
	return nil
}

func (m *Module) Health(ctx context.Context) error {
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

func (m *Module) allow(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.buckets[key]
	if !ok {
		m.buckets[key] = &bucket{
			tokens:   float64(m.burst) - 1,
			lastTick: time.Now(),
		}
		return true
	}

	now := time.Now()
	elapsed := now.Sub(b.lastTick).Seconds()
	b.lastTick = now

	b.tokens = math.Min(b.tokens+elapsed*m.rate, float64(m.burst))
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
