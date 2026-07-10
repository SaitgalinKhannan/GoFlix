import type { TorrentStatus } from "./types";

// API base URL: explicit override via env, otherwise same-origin (nginx proxies /api).
export const API_BASE =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") || "";

const ACCESS_KEY = "goflix_access";
const REFRESH_KEY = "goflix_refresh";

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

// ---- token storage ----

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ACCESS_KEY);
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_KEY);
}

export function setTokens(pair: TokenPair): void {
  localStorage.setItem(ACCESS_KEY, pair.accessToken);
  localStorage.setItem(REFRESH_KEY, pair.refreshToken);
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export function hasTokens(): boolean {
  return !!getAccessToken();
}

// ---- auth ----

export async function login(
  username: string,
  password: string,
): Promise<TokenPair> {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `login failed: ${res.status}`);
  }
  const pair = (await res.json()) as TokenPair;
  setTokens(pair);
  return pair;
}

export async function refresh(): Promise<TokenPair> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) throw new Error("no refresh token");
  const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
  });
  if (!res.ok) {
    clearTokens();
    throw new Error(`refresh failed: ${res.status}`);
  }
  const pair = (await res.json()) as TokenPair;
  setTokens(pair);
  return pair;
}

// ---- request wrapper with auto-refresh on 401 ----

async function authedFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const access = getAccessToken();
  const headers = new Headers(init.headers);
  if (access) headers.set("Authorization", `Bearer ${access}`);

  let res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (res.status !== 401) return res;

  // Access token expired — try to refresh once and replay.
  try {
    await refresh();
  } catch {
    return res; // refresh failed; return the 401 so the caller can log out.
  }
  headers.set("Authorization", `Bearer ${getAccessToken()}`);
  return fetch(`${API_BASE}${path}`, { ...init, headers });
}

// ---- torrents ----

export async function listTorrents(): Promise<TorrentStatus[]> {
  const res = await authedFetch("/api/v1/torrents");
  if (!res.ok) throw new Error(`list failed: ${res.status}`);
  return res.json();
}

export async function getTorrent(id: string): Promise<TorrentStatus> {
  const res = await authedFetch(`/api/v1/torrents/${id}`);
  if (!res.ok) throw new Error(`get failed: ${res.status}`);
  return res.json();
}

export async function addMagnet(magnet: string): Promise<TorrentStatus> {
  const form = new FormData();
  form.append("magnet", magnet);
  const res = await authedFetch("/api/v1/torrents", { method: "POST", body: form });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `add failed: ${res.status}`);
  }
  return res.json();
}

export async function addTorrentFile(file: File): Promise<TorrentStatus> {
  const form = new FormData();
  form.append("file", file);
  const res = await authedFetch("/api/v1/torrents", { method: "POST", body: form });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `add failed: ${res.status}`);
  }
  return res.json();
}

export async function deleteTorrent(
  id: string,
  deleteFiles: boolean,
): Promise<void> {
  const url = `/api/v1/torrents/${id}${deleteFiles ? "?deleteFiles=true" : ""}`;
  const res = await authedFetch(url, { method: "DELETE" });
  if (!res.ok && res.status !== 204) throw new Error(`delete failed: ${res.status}`);
}

export async function startSeeding(id: string): Promise<void> {
  const res = await authedFetch(`/api/v1/torrents/${id}/start`, { method: "POST" });
  if (!res.ok) throw new Error(`start failed: ${res.status}`);
}

export async function stopSeeding(id: string): Promise<void> {
  const res = await authedFetch(`/api/v1/torrents/${id}/stop`, { method: "POST" });
  if (!res.ok) throw new Error(`stop failed: ${res.status}`);
}

export async function recheckTorrent(id: string): Promise<void> {
  const res = await authedFetch(`/api/v1/torrents/${id}/recheck`, { method: "POST" });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `recheck failed: ${res.status}`);
  }
}
