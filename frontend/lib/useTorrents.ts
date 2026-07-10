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
      headers: { Authorization: `Bearer ${getAccessToken()}` },
      signal: ctrl.signal,
      openWhenHidden: true,

      async onopen(res) {
        if (res.ok) {
          setConnected(true);
          stopPolling();
          return;
        }
        if (res.status === 401) {
          // Try a token refresh; if it fails, surface the auth error.
          try {
            await refreshTokens();
            throw new Error("retry"); // reopen with the fresh token
          } catch {
            onAuthError?.();
          }
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
      },
      onclose() {
        if (stopped) return;
        setConnected(false);
        startPolling();
      },
    }).catch(() => {
      if (!stopped) startPolling();
    });

    return () => {
      stopped = true;
      stopPolling();
      ctrl.abort();
    };
  }, [enabled, onAuthError]);

  return { torrents, connected, refresh };
}
