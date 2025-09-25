```mermaid
sequenceDiagram
    participant User as 👤 User (React UI)
    participant UI as 💻 React UI (WebSocket)
    participant BE as 🧠 Backend (Go API)
    participant RQ as ⚙️ River queue
    participant REDIS as 📡 Redis
    participant WS as 🌐 WebSocket Server

        User->>UI: Adds Cloud Provider
        UI->>BE: POST /api/providers
        BE->>RQ: Start Riverqueue Job (fetch metadata)

        Note right of RQ: ~10-15 mins\nfetches clusters, etc.

        RQ-->>BE: ActivationStatus = ACTIVE
        RQ->>REDIS: Publish to\nfutura.org.{orgID}.provider.{providerID}.activated

        REDIS->>WS: Message received
        WS->>UI: Send WebSocket message\n{ type: "provider_activated", providerId }

        UI->>User: 🔔 Show toast\n"Your clusters are ready!"
        UI->>BE: (optional) refetch clusters list
```
