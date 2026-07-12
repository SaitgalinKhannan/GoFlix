"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { fetchEventSource } from "@microsoft/fetch-event-source";
import { API_BASE, getAccessToken, listTorrents, refresh as refreshTokens } from "./api";
import type { TorrentStatus } from "./types";

/**
 * useTorrents subscribes to the live SSE status stream and keeps the torrent
 * list in state. Falls back to polling if SSE fails.
 */
export function useTorrents(enabled: boolean, onAuthError?: () => void) {
  const [torrents, setTorrents] = useState<TorrentStatus[]>([]);
  const [connected, setConnected] = useState(false);
  const refreshFn = useRef<() => void>(() => {});

  const refresh = useCallback(async () => {
    try {
      const list = await listTorrents();
      setTorrents(list);
    } catch {
      /* ignore transient polling errors */
    }
  }, []);

  refreshFn.current = refresh;

  useEffect(() => {
    if (!enabled) return;

    let stopped = false;
    let authFailed = false;
    let pollTimer: ReturnType<typeof setInterval> | null = null;
    const ctrl = new AbortController();

    const startPolling = () => {
      if (pollTimer) return;
      pollTimer = setInterval(() => refreshFn.current(), 5000);
    };
    const stopPolling = () => {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    };

    refreshFn.current();

    fetchEventSource(`${API_BASE}/api/v1/events`, {
      method: "GET",
      signal: ctrl.signal,
      openWhenHidden: true,
      // fetch-event-source keeps its original headers between retries. Read
      // the token for every connection, otherwise an SSE stream cannot recover
      // after the access token expires while the page is open.
      fetch(input, init) {
        const headers = new Headers(init?.headers);
        const accessToken = getAccessToken();
        if (accessToken) {
          headers.set("Authorization", `Bearer ${accessToken}`);
        } else {
          headers.delete("Authorization");
        }
        return fetch(input, { ...init, headers });
      },

      async onopen(res) {
        if (res.ok) {
          setConnected(true);
          stopPolling();
          void refreshFn.current();
          return;
        }
        if (res.status === 401) {
          // Refresh first; the custom fetch above supplies the new token to
          // the library's next connection attempt.
          try {
            await refreshTokens();
          } catch {
            authFailed = true;
            onAuthError?.();
          }
          throw new Error("SSE token refresh required");
        }
        throw new Error(`SSE open failed: ${res.status}`);
      },
      onmessage(ev) {
        if (ev.event === "status") {
          try {
            setTorrents(JSON.parse(ev.data) as TorrentStatus[]);
          } catch {
            /* ignore malformed payloads */
          }
        }
      },
      onerror() {
        if (stopped) return;
        setConnected(false);
        startPolling();
        if (authFailed) {
          throw new Error("SSE authentication failed");
        }
        return 3000;
      },
      onclose() {
        if (stopped) return;
        setConnected(false);
        startPolling();
        // A cleanly closed HTTP stream is otherwise treated as final by the
        // library, leaving the UI in polling mode forever after a reconnect.
        throw new Error("SSE connection closed");
      },
    }).catch(() => {
      if (!stopped && !authFailed) startPolling();
    });

    return () => {
      stopped = true;
      stopPolling();
      ctrl.abort();
    };
  }, [enabled, onAuthError]);

  return { torrents, connected, refresh };
}
