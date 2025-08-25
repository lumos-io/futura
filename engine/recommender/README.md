# Recommender

## Go Migrate tool

This is for the Analytics service as I don't have a stable `gorm` library for ClickHouse. There is a gorm plugin but not very stable. Instead, we use `golang-migrate` tool to migrate up or down the versions of the DB.

```bash
brew install golang-migrate
export CLICKHOUSE_URL='clickhouse://localhost:9000?username=user&password=password&database=events&x-multi-statement=true'
migrate -database ${CLICKHOUSE_URL} -path db/migrations up
```
