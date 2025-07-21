# Frontend

To add the feature flag, you need to just do the below

```typescript
import { useFlag } from "@unleash/proxy-client-react";

// ...
const enabled = useFlag("<name_of_flag>");
if (enabled) {
  console.log("feature flag enabled");
}
```
