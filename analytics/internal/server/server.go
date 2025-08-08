package server

import (
	"context"

	"github.com/opisvigilant/futura/analytics/internal/config"
	pbsvc "github.com/opisvigilant/futura/proto/gen/analytics"
)

type AnalyticsServer struct {
	pbsvc.UnimplementedAnalyticsServiceServer
}

func NewAnalyticsServer(config *config.Configuration) (*AnalyticsServer, error) {
	return &AnalyticsServer{}, nil
}

func (a *AnalyticsServer) Close() error {
	return nil
}

func (a *AnalyticsServer) GetEvents(ctx context.Context, req *pbsvc.GetEventsByClusterIdRequest) (*pbsvc.GetEventsResponse, error) {
	return nil, nil
}
