package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Client struct {
	conn driver.Conn
}

func New(addrs []string, username, password, database string, debug bool) (*Client, error) {
	conn, err := ch.Open(&ch.Options{
		Addr: addrs,
		Auth: ch.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Debug: debug,
		Debugf: func(format string, v ...any) {
			fmt.Printf(format+"\n", v...)
		},
		Compression: &ch.Compression{
			Method: ch.CompressionLZ4,
		},
		TLS: &tls.Config{
			InsecureSkipVerify: debug,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, err
	}

	return &Client{
		conn: conn,
	}, nil
}
