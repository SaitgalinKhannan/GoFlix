#!/bin/bash
# Convenience wrapper to build & run the torrent-service locally.
set -e

go build -o server ./cmd/server
./server
