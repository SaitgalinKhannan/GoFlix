"use client";

declare global {
  interface Window {
    __TAURI__?: {
      core?: {
        invoke<T>(command: string, args?: Record<string, unknown>): Promise<T>;
      };
    };
  }
}

export type ServiceState = "running" | "stopped" | "error";

export type DesktopStatus = {
  backend: ServiceState;
  jellyfin: ServiceState;
  apiUrl: string;
  jellyfinUrl: string;
  startupError?: string | null;
};

export function isDesktop(): boolean {
  return typeof window !== "undefined" && !!window.__TAURI__?.core?.invoke;
}

export function desktopInvoke<T>(
  command: string,
  args?: Record<string, unknown>,
): Promise<T> {
  const nativeInvoke = window.__TAURI__?.core?.invoke;
  if (!nativeInvoke) return Promise.reject(new Error("Desktop API is unavailable"));
  return nativeInvoke<T>(command, args);
}
