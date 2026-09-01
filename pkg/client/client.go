package client

import (
	"context"

	"github.com/Muxcore-Media/core/pkg/contracts"
	ratelimitv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/ratelimit/v1"
	"google.golang.org/grpc"
)

// Client implements contracts.RateLimiterProvider over RateLimitService gRPC.
type Client struct {
	rpc ratelimitv1.RateLimitServiceClient
}

// New returns a RateLimiterProvider backed by conn.
func New(conn grpc.ClientConnInterface) *Client {
	return &Client{rpc: ratelimitv1.NewRateLimitServiceClient(conn)}
}

var _ contracts.RateLimiterProvider = (*Client)(nil)

// Allow checks whether a request from key should proceed. RPC errors fail open.
func (c *Client) Allow(ctx context.Context, key string) bool {
	resp, err := c.rpc.Allow(ctx, &ratelimitv1.AllowRequest{Key: key})
	if err != nil {
		return true
	}
	return resp.GetAllowed()
}

// Enabled reports whether rate limiting is active on the sidecar. RPC errors return false.
func (c *Client) Enabled() bool {
	resp, err := c.rpc.Enabled(context.Background(), &ratelimitv1.EnabledRequest{})
	if err != nil {
		return false
	}
	return resp.GetEnabled()
}
