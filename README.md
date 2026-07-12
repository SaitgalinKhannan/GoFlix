# GoFlix

Download torrents from your phone/PC and stream them to your TV via Jellyfin.

```
Phone / PC (browser)
   │  HTTPS via Cloudflare Tunnel
   ▼
frontend (nginx, Next.js static export)
   │  fetch / SSE, JWT Bearer header
   ▼
torrent-service (Go, anacrolix/torrent)
   │  writes files
   ▼
shared volume (goflix-data)
   ▲
   │  reads as a library
jellyfin
   │
   ▼
TV (Jellyfin app)
```

When a torrent finishes downloading, `torrent-service` calls
`POST {JELLYFIN_URL}/Library/Refresh` so Jellyfin picks up the new file.

## Stack

- **torrent-service** — Go + [`anacrolix/torrent`](https://github.com/anacrolix/torrent). Downloads torrents, exposes a REST + SSE API, persists state to JSON.
- **jellyfin** — official Docker image, shares the downloads volume read-only as a library.
- **frontend** — Next.js (TypeScript, Tailwind), static export served by nginx. Proxies `/api` to torrent-service.
- **orchestration** — Docker Compose.

## Quick start (Docker Compose)

1. **Configure** — copy and edit the environment file:
   ```bash
   cp .env.example .env
   # set JWT_SECRET (openssl rand -hex 32), AUTH_USERNAME, AUTH_PASSWORD, and JELLYFIN_TOKEN
   ```

2. **Generate a Jellyfin API key** — Jellyfin → Dashboard → API Keys → "+". Put the token in `JELLYFIN_TOKEN`.

3. **Start everything**:
   ```bash
   docker compose up -d --build
   ```

4. **Open the UI** at `http://<server>:8080` (set `FRONTEND_PORT` to change it), sign in with your `AUTH_USERNAME`/`AUTH_PASSWORD`, and paste a magnet link or drop a `.torrent` file.
5. **Point Jellyfin at the media** (first run only): open Jellyfin at `http://<server>:8096`, finish setup, and add a **Media Library** with the folder `/data/torrents` — that's where downloads land (the `goflix-data` volume, mounted read-only).

## Desktop shell (Tauri)

`desktop/` contains the lightweight native shell for Linux desktops. It uses the
system WebView rather than Electron. In the desktop app, **Choose folder** opens
the native folder picker, writes an absolute `DATA_DIR` to the project `.env`,
creates its required subfolders, and restarts `docker compose`.

Changing `DATA_DIR` does not move existing downloads. Move them manually before
switching folders if they must remain available to Jellyfin.

For development:

```bash
cd desktop
npm install
npm run dev
```

## Exposing the UI over the internet (Cloudflare Tunnel)

The `frontend` container (nginx) is the single entrypoint: it serves the UI and
reverse-proxies `/api` to torrent-service. A `cloudflared` container exposes it on
a public HTTPS hostname — no ports forwarded. The tunnel reaches `frontend` over the
Docker network, so the public hostname sees the UI and the API as one origin.

Setup is one variable:

1. In the Cloudflare dashboard: **Zero Trust → Networks → Tunnels → Create a tunnel**
   (type *Cloudflared*).
2. Add a **Public Hostname** to it, e.g. `goflix.yourdomain.com` →
   Service `HTTP` / URL `frontend:80`.
3. Copy the tunnel token into `.env`:
   ```env
   TUNNEL_TOKEN=eyJh…your…token…
   ```
4. Start with the tunnel profile:
   ```bash
   docker compose --profile tunnel up -d --build
   ```
5. Open `https://goflix.yourdomain.com`.

> Without `--profile tunnel`, `cloudflared` stays stopped and GoFlix is reachable
> on the LAN at `http://<server>:8082` (or your `FRONTEND_PORT`).

## Configuration

All config is environment-driven (see `.env.example`):

| Variable                  | Default                          | Description                                              |
| ------------------------- | -------------------------------- | ------------------------------------------------------- |
| `PORT`                    | `8081`                           | torrent-service listen port                             |
| `DATA_DIR`                | `./data`                         | Host folder bind-mounted into the containers (downloads + state). Use an absolute path. |
| `TORRENTS_DIR`            | `/app/data/torrents`            | Where downloaded media is written (inside the container) |
| `PIECE_COMPLETION_DIR`    | `/app/data/torrent_data`        | Legacy setting (completion is rebuilt by startup recheck) |
| `TORRENTS_STATES_FILE`    | `/app/data/torrent_states.json` | Persisted torrent list (for resume after restart)        |
| `SPEED_LIMITS_FILE`       | `/app/data/speed_limits.json`   | Persisted global download/upload limits                  |
| `DOWNLOAD_RATE_LIMIT_KIB` | `0`                              | Initial global download limit in KiB/s (`0` = unlimited) |
| `UPLOAD_RATE_LIMIT_KIB`   | `0`                              | Initial global upload limit in KiB/s (`0` = unlimited)   |
| `JWT_SECRET`              | _(required)_                     | Secret signing JWTs (`openssl rand -hex 32`)             |
| `AUTH_USERNAME`           | _(required)_                     | Username for the UI login                                |
| `AUTH_PASSWORD`           | _(required)_                     | Password for the UI login (hashed at startup)            |
| `JELLYFIN_URL`            | `http://jellyfin:8096`          | Jellyfin base URL for the refresh hook                   |
| `JELLYFIN_TOKEN`          | _(empty)_                        | Jellyfin API key (`X-Emby-Token`). Empty disables hook   |
| `MAX_CONCURRENT_DOWNLOADS`| `3`                              | Cap on simultaneous downloads (extras are paused)        |
| `TUNNEL_TOKEN`            | _(empty)_                        | Cloudflare tunnel token. Set it → UI is public on HTTPS  |

## API

All `/api/v1` endpoints (except `/auth/*`) require a JWT access token in the
`Authorization: Bearer <token>` header. Obtain one via login.

```
POST   /api/v1/auth/login           { username, password } → { accessToken, refreshToken, expiresIn }
POST   /api/v1/auth/refresh         { refreshToken } → new token pair
POST   /api/v1/torrents             multipart/form-data: "magnet" (string) OR "file" (.torrent)
                                    → 201 TorrentStatus
GET    /api/v1/torrents              → 200 TorrentStatus[]
GET    /api/v1/torrents/{id}         → 200 TorrentStatus (with files[])
DELETE /api/v1/torrents/{id}         ?deleteFiles=true → 204
POST   /api/v1/torrents/{id}/start   → 200   (resume downloading/seeding)
POST   /api/v1/torrents/{id}/stop    → 200   (stop, files preserved)
POST   /api/v1/torrents/{id}/recheck → 202   (queue full piece hash verification)
GET    /api/v1/settings/speed-limits  → 200   ({ download, upload } bytes/s)
PUT    /api/v1/settings/speed-limits  → 200   (update and persist global limits)
GET    /api/v1/events                Server-Sent Events: live TorrentStatus[] stream
```

Authentication uses JWT: login returns a short-lived **access token** (15 min)
and a long-lived **refresh token** (30 days). The frontend stores both and
auto-refreshes on a 401.

Unauthenticated helpers:

```
GET    /api/files/tree              file tree under the downloads dir
GET    /api/files?path=<path>       list a directory
GET    /api/health                  → 200 OK
```

```ts
type TorrentStatus = {
  id: string;
  infoHash: string;
  name: string;
  status: "metadata" | "queued" | "downloading" | "seeding" | "completed" | "paused" | "error";
  progress: number;        // 0..1
  downloadSpeed: number;   // bytes/s
  uploadSpeed: number;     // bytes/s
  seeders: number;         // currently connected peers with all pieces
  leechers: number;        // currently connected peers still downloading
  size: number;            // bytes
  downloaded: number;      // bytes
  addedAt: string;         // ISO 8601
  completedAt?: string;
  error?: string;
  files?: { path: string; size: number }[];
};
```

### Examples

```bash
# Login → save the access token
TOKEN=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r .accessToken)

# Add via magnet
curl -X POST http://localhost:8081/api/v1/torrents \
  -H "Authorization: Bearer $TOKEN" \
  -F "magnet=magnet:?xt=urn:btih:..."

# Add via .torrent file
curl -X POST http://localhost:8081/api/v1/torrents \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@movie.torrent"

# List
curl http://localhost:8081/api/v1/torrents -H "Authorization: Bearer $TOKEN"

# Delete and remove files
curl -X DELETE "http://localhost:8081/api/v1/torrents/<id>?deleteFiles=true" \
  -H "Authorization: Bearer $TOKEN"
```

## Local development

**torrent-service:**
```bash
# Requires JWT_SECRET, AUTH_USERNAME, AUTH_PASSWORD in .env (defaults in the dev .env).
go run ./cmd/server
```

**frontend:**
```bash
cd frontend
npm install
npm run dev             # http://localhost:3000 (proxies /api via NEXT_PUBLIC_API_URL)
```
Point the UI at the local backend with `NEXT_PUBLIC_API_URL=http://localhost:8081 npm run dev`,
or leave it unset and run behind the nginx proxy in Compose.

## Project structure

```
cmd/server/main.go             # entry point
configs/                       # env-driven config
internal/
  app/torrent/                 # client, state, service, SSE hub, event handler
  app/web/                     # HTTP server, routes, JWT middleware
  app/auth/                    # JWT issue/verify, bcrypt credentials
  app/web/handlers/            # REST + SSE handlers
  app/filesystem/              # file browsing + path-safety
  pkg/{httphelpers,filehelpers}/
frontend/                      # Next.js app
Dockerfile                     # torrent-service image (no ffmpeg)
docker-compose.yml             # torrent-service + jellyfin + frontend
```
