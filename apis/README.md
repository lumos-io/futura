# APIs

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
