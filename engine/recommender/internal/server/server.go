package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	"github.com/opisvigilant/futura/engine/recommender/internal/routines"
	pbeng "github.com/opisvigilant/futura/proto/gen/engine"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RecommenderServer struct {
	pbeng.UnimplementedRecommendationServiceServer

	mdRecommender *routines.MultiDimensionRecommender
}

func NewRecommenderServer(config *config.Configuration) (*RecommenderServer, error) {
	mdr, err := routines.NewMultiDimensionRecommender(config)
	if err != nil {
		return nil, err
	}
	return &RecommenderServer{
		mdRecommender: mdr,
	}, nil
}

func (a *RecommenderServer) Close() error {
	return nil
}

func (a *RecommenderServer) GetRecommendation(ctx context.Context, req *pbeng.RecommendationRequest) (*pbeng.RecommendationResponse, error) {
	ap, err := a.mdRecommender.GetActionPlan(req.App)
	if err != nil {
		return nil, err
	}
	return &pbeng.RecommendationResponse{
		Plan:            ap,
		DecisionId:      uuid.NewString(),
		Confidence:      0.9,
		ModelVersion:    "v0.0.1",
		EffectivePolicy: &pbeng.SafetyPolicy{},
	}, nil
}

// Operator posts execution outcome/telemetry for learning & audit
func (a *RecommenderServer) ReportExecutionOutcome(ctx context.Context, req *pbeng.ExecutionOutcome) (*emptypb.Empty, error) {
	return nil, nil
}
