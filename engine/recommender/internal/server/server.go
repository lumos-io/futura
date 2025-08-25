package server

import (
	"context"

	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	pbsvc "github.com/opisvigilant/futura/proto/gen/engine"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RecommenderServer struct {
	pbsvc.UnimplementedRecommendationServiceServer
}

func NewRecommenderServer(config *config.Configuration) (*RecommenderServer, error) {
	return &RecommenderServer{}, nil
}

func (a *RecommenderServer) Close() error {
	return nil
}

func (a *RecommenderServer) GetRecommendation(ctx context.Context, req *pbsvc.RecommendationRequest) (*pbsvc.RecommendationResponse, error) {
	return nil, nil
}

// Operator posts execution outcome/telemetry for learning & audit
func (a *RecommenderServer) ReportExecutionOutcome(context.Context, *pbsvc.ExecutionOutcome) (*emptypb.Empty, error) {
	return nil, nil
}
