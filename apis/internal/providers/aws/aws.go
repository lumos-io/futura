package awsprovider

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
	"gorm.io/datatypes"
)

const (
	AWS_ACCOUNT           string = "Account"
	AWS_ACCESS_KEY        string = "AccessKey"
	AWS_SECRET_ACCESS_KEY string = "SecretAccessKey"
	AWS_REGION            string = "Region"
)

type AWSProvider struct {
	account string
	cfg     aws.Config
	client  *eks.Client
}

func New(creds map[string]string) (*AWSProvider, error) {
	// Manually construct the AWS config with static credentials
	account, cfg, err := buildAWSConfiguration(creds)
	if err != nil {
		return nil, err
	}
	client := eks.NewFromConfig(*cfg)
	return &AWSProvider{account: *account, cfg: *cfg, client: client}, nil
}

func buildAWSConfiguration(creds map[string]string) (*string, *aws.Config, error) {
	account, ok := creds[AWS_ACCOUNT]
	if !ok {
		return nil, nil, errors.New("AWS_ACCOUNT parameter not found")
	}
	region, ok := creds[AWS_REGION]
	if !ok {
		return nil, nil, errors.New("AWS_REGION parameter not found")
	}
	key, ok := creds[AWS_ACCESS_KEY]
	if !ok {
		return nil, nil, errors.New("AWS_ACCESS_KEY parameter not found")
	}
	secret, ok := creds[AWS_SECRET_ACCESS_KEY]
	if !ok {
		return nil, nil, errors.New("AWS_SECRET_ACCESS_KEY parameter not found")
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key, secret, ""),
		),
	)
	if err != nil {
		return nil, nil, err
	}
	return &account, &cfg, nil
}

func (a *AWSProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

func (a *AWSProvider) FetchClusters(ctx context.Context) ([]string, error) {
	var allClusters []string
	var nextToken *string = nil
	for {
		resp, err := a.client.ListClusters(ctx, &eks.ListClustersInput{
			Include:   []string{"all"},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		allClusters = append(allClusters, resp.Clusters...)

		if resp.NextToken == nil {
			break // no more pages
		}
		nextToken = resp.NextToken
	}

	return allClusters, nil
}

func (a *AWSProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	resp, err := a.client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(clusterID),
	})
	if err != nil {
		return nil, err
	}
	cluster := resp.Cluster
	m := &models.EKSClusterMetadata{
		EKSClusterName:   aws.ToString(cluster.Name),
		Status:           string(cluster.Status),
		Version:          aws.ToString(cluster.Version),
		Endpoint:         aws.ToString(cluster.Endpoint),
		Arn:              aws.ToString(cluster.Arn),
		ClusterCreatedAt: cluster.CreatedAt,
		EKSClusterID:     aws.ToString(cluster.Id),
		PlatformVersion:  aws.ToString(cluster.PlatformVersion),
		Tags:             datatypes.JSONMap(utils.ConvertStringMapToInterfaceMap(cluster.Tags)),
	}
	return &models.ClusterMetadata{
		EKSMetadata: m,
	}, nil
}
