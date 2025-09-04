package analytics

import (
	"context"

	"github.com/opisvigilant/futura/apis/internal/config"
	"google.golang.org/grpc"

	pban "github.com/opisvigilant/futura/proto/gen/analytics"
)

type Client struct {
	pban.AnalyticsServiceClient
}

func New(config *config.Configuration) (*Client, error) {
	var opts []grpc.DialOption
	conn, err := grpc.NewClient(config.Analytics.Endpoint, opts...)
	if err != nil {
		return nil, err
	}
	client := pban.NewAnalyticsServiceClient(conn)
	return &Client{client}, nil
}

func (c *Client) GetEvents() {
	resp, err := c.AnalyticsServiceClient.GetEvents(context.Background(), &pban.GetEventsByClusterIdRequest{})
	if err != nil {

	}
	resp.GetEvents()
}

func (c *Client) Close() error {
	return c.Close()
}
