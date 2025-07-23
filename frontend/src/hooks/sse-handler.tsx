import { useEffect, useRef, useState } from "react";

interface UseSSEOptions<T> {
  event?: string;
  onMessage?: (data: T) => void;
  onError?: (error: Event) => void;
  withCredentials?: boolean;
}

export function useSSE<T = unknown>(
  url: string,
  options?: UseSSEOptions<T>
): {
  latest: T | null;
  connected: boolean;
} {
  const [latest, setLatest] = useState<T | null>(null);
  const [connected, setConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!url) return;

    const source = new EventSource(url, {
      withCredentials: options?.withCredentials ?? false,
    });
    eventSourceRef.current = source;

    const onOpen = () => setConnected(true);
    const onError = (err: Event) => {
      setConnected(false);
      options?.onError?.(err);
    };
    const onMessage = (event: MessageEvent) => {
      try {
        const data: T = JSON.parse(event.data);
        setLatest(data);
        options?.onMessage?.(data);
      } catch (err) {
        console.error("Failed to parse SSE data", err);
      }
    };

    source.addEventListener("open", onOpen);
    source.addEventListener("error", onError);

    if (options?.event) {
      source.addEventListener(options.event, onMessage);
    } else {
      source.addEventListener("message", onMessage);
    }

    return () => {
      source.close();
    };
  }, [url, options?.event]);

  return { latest, connected };
}
