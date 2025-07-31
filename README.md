# Goflix Server

High-performance torrent streaming backend built with Go. Download torrents and stream content via REST API and WebSocket connections.

## Features

- 🔥 **Torrent Client** - Add torrents via magnet links
- 📊 **Real-time Updates** - Live download progress via WebSocket
- 📁 **File System API** - Browse downloaded content
- 🛡️ **Path Traversal Protection** - Secure file access
- 🚀 **Graceful Shutdown** - Clean torrent client closure
- 📡 **CORS Support** - Ready for web frontend integration

## Tech Stack

- **Language**: Go 1.24
- **Torrent**: [anacrolix/torrent](https://github.com/anacrolix/torrent)
- **HTTP Router**: [go-chi/chi](https://github.com/go-chi/chi)
- **WebSocket**: [gorilla/websocket](https://github.com/gorilla/websocket)

## Project Structure

```
├── cmd/server/main.go           # Application entry point
├── internal/
│   ├── app/
│   │   ├── filesystem/          # File system operations
│   │   ├── torrent/            # Torrent client logic
│   │   └── web/handlers/       # HTTP handlers
│   └── pkg/httplog/            # HTTP middleware
├── torrents/                   # Downloaded content (auto-created)
└── torrent_data/              # Torrent metadata (auto-created)
```

## Quick Start

1. **Clone and install dependencies**:
```bash
git clone github.com/SaitgalinKhannan/GoFlix
cd GoFlix
go mod download
```

2. **Run the server**:
```bash
go run cmd/server/main.go
```

3. **Server starts on** `http://localhost:8080`

## API Endpoints

### REST API
- `POST /api/torrents/add` - Add torrent via magnet link
- `GET /api/torrents/all` - Get all torrents with progress
- `GET /api/files/tree` - Get complete file tree
- `GET /api/files?path=<path>` - Get files in specific directory
- `GET /api/health` - Health check

### WebSocket
- `GET /ws` - Real-time torrent progress updates

## API Examples

**Add a torrent**:
```bash
curl -X POST http://localhost:8080/api/torrents/add \
  -H "Content-Type: application/json" \
  -d '{"source": "magnet:?xt=urn:btih:..."}'
```

**Get torrent status**:
```bash
curl http://localhost:8080/api/torrents/all
```

**Browse files**:
```bash
curl http://localhost:8080/api/files?path=/Movie.Name
```

## Configuration

The server uses these directories:
- `./torrents/` - Downloaded torrent content
- `./torrent_data/` - Torrent piece completion data

Both directories are created automatically on first run.

## Features in Detail

**Graceful Shutdown**: Server properly closes all torrent connections on SIGTERM/SIGINT

**Security**: Path traversal protection prevents accessing files outside the torrents directory

**Error Handling**: Comprehensive error handling and logging throughout the application

**Performance**: Efficient file system operations and optimized torrent handling

---

*Built for seamless integration with the Goflix SvelteKit frontend*