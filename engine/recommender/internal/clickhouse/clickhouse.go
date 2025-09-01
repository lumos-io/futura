package clickhouse

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
)

type Client struct {
	clickhouse.Conn
}

func New(config *config.Clickhouse) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: config.Servers,
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		conn,
	}, nil
}
