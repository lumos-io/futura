import { CloudProviderConnection } from "@/models/cloud-provider";
import { useEffect, useRef } from "react";

type UseMultiWsOptions = {
    onUpdate: (update: CloudProviderConnection) => void;
    wsBaseUrl: string; // e.g. ws://localhost:8080/ws
};

/**
 * Hook that manages WebSocket connections for multiple providers
 * Automatically opens/closes sockets as provider IDs change
 */
export function useMultiWsConnections(
    providerIds: string[],
    { onUpdate, wsBaseUrl }: UseMultiWsOptions
) {
    // Keep refs to WebSocket connections
    const wsConnections = useRef<Record<string, WebSocket>>({});

    useEffect(() => {
        // Open WS for newly added provider IDs
        providerIds.forEach((id) => {
            if (!wsConnections.current[id]) {
                const wsUrl = `${wsBaseUrl}/${id}`;
                const ws = new WebSocket(wsUrl);

                ws.onopen = () => {
                    console.log("WebSocket opened for id", id);
                    // safe to send now or mark ready state
                };

                ws.onmessage = (event) => {
                    try {
                        const data: CloudProviderConnection = JSON.parse(event.data);
                        onUpdate(data);
                    } catch (e) {
                        console.error("Failed to parse WS message:", e);
                    }
                };

                ws.onerror = (err) => console.error(`WS error for ${id}:`, err);

                ws.onclose = () => {
                    console.log(`WS closed for provider ${id}`);
                    delete wsConnections.current[id];
                };

                wsConnections.current[id] = ws;
            }
        });

        // Close WS for provider IDs no longer present
        Object.keys(wsConnections.current).forEach((id) => {
            if (!providerIds.includes(id)) {
                wsConnections.current[id].close();
                delete wsConnections.current[id];
            }
        });

        // Cleanup all sockets on unmount
        return () => {
            Object.values(wsConnections.current).forEach((ws) => ws.close());
            wsConnections.current = {};
        };
    }, [providerIds, onUpdate, wsBaseUrl]);
}