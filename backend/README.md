# Backend

## APIs feature flag

To add the feature flag check, do the below in the code

```go
import (
    "github.com/Unleash/unleash-client-go/v4"
)

// in the function where needed
func banana() {
    if unleash.IsEnabled("<name_of_flag>") {
        // do what it is necessary
    }
}
```

## Clickhouse migrations

```bash
export CLICKHOUSE_URL='clickhouse://localhost:9000?username=user&password=password&database=futura&x-multi-statement=true'
migrate -database ${CLICKHOUSE_URL} -path internal/migrations/clickhouse/migrations up
```
