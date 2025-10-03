package analytics

import (
	"context"

	"github.com/opisvigilant/futura/apis/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pban "github.com/opisvigilant/futura/proto/gen/analytics"
	pbtl "github.com/opisvigilant/futura/proto/gen/telemetry"
)

type Client struct {
	pban.AnalyticsServiceClient
	conn *grpc.ClientConn
}

func New(config *config.Configuration) (*Client, error) {
	conn, err := grpc.NewClient(config.Analytics.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pban.NewAnalyticsServiceClient(conn)
	return &Client{
		AnalyticsServiceClient: client,
		conn:                   conn,
	}, nil
}

func (c *Client) GetEvents(req *pban.GetEventsByClusterIdRequest) ([]*pbtl.KubernetesEvent, error) {
	resp, err := c.AnalyticsServiceClient.GetEvents(context.Background(), req)
	if err != nil {
		return nil, err
	}
	events := resp.GetEvents()
	return events, nil
}

func (c *Client) GetNodes(req *pban.GetNodesByClusterIdRequest) (*pban.GetNodesResponse, error) {
	resp, err := c.AnalyticsServiceClient.GetNodes(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetClusterConfig(req *pban.GetClusterConfigRequest) (*pban.ClusterConfigResponse, error) {
	resp, err := c.AnalyticsServiceClient.GetClusterConfig(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetServices(req *pban.GetServicesByClusterIdRequest) (*pban.GetServicesResponse, error) {
	resp, err := c.AnalyticsServiceClient.GetServices(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetOverviewMetrics(req *pban.GetOverviewMetricsRequest) (*pban.OverviewMetrics, error) {
	resp, err := c.AnalyticsServiceClient.GetOverviewMetrics(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
