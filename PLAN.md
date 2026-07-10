# Torrent Downloader + Jellyfin — план проекта

## Цель
Открыть страницу с телефона / пк в браузере → кинуть magnet-ссылку или .torrent файл → сервер качает фильм → Jellyfin подхватывает файл → смотришь на ТВ.

## Стек
- **torrent-service**: Go + `anacrolix/torrent` — качает торренты, отдаёт REST/SSE API
- **jellyfin**: официальный docker-образ, не трогаем логику, только шарим volume
- **frontend**: Next.js TypeScript, отдельный контейнер (nginx со статикой)
- **orchestration**: Docker Compose рядом с существующими сервисами на MainRenServer
- **сеть**: Nginx + Cloudflare Tunnel

## Архитектура

```
Телефон (браузер, React SPA)
   │  HTTPS через Cloudflare Tunnel
   ▼
frontend (nginx, статика)
   │  fetch/SSE, заголовок X-API-Key
   ▼
torrent-service (Go, anacrolix/torrent)
   │  пишет файлы
   ▼
shared volume (media-downloads)
   ▲
   │  читает как library
jellyfin
   │
   ▼
ТВ (Jellyfin app / Tizen)
```

После завершения скачивания torrent-service дёргает `POST {JELLYFIN_URL}/Library/Refresh` с заголовком `X-Emby-Token`.

## API контракт torrent-service

Все эндпоинты за заголовком `X-API-Key: <secret>` (значение из env, сверяется мидлварой).

```
POST   /api/v1/torrents
  multipart/form-data: либо поле "magnet" (string), либо "file" (.torrent)
  → 201 { id, infoHash, name, status: "metadata" }

GET    /api/v1/torrents
  → 200 [ TorrentStatus, ... ]

GET    /api/v1/torrents/{id}
  → 200 TorrentStatus (с files[])

DELETE /api/v1/torrents/{id}?deleteFiles=true
  → 204

GET    /api/v1/events   (Server-Sent Events)
  → поток TorrentStatus при любом изменении прогресса/статуса
```

```ts
type TorrentStatus = {
  id: string
  infoHash: string
  name: string
  status: "metadata" | "downloading" | "seeding" | "completed" | "error"
  progress: number        // 0..1
  downloadSpeed: number    // bytes/s
  uploadSpeed: number
  size: number             // bytes
  downloaded: number        // bytes
  addedAt: string           // ISO 8601
  completedAt?: string
  error?: string
  files?: { path: string; size: number }[]
}
```

## Ключевые технические решения (зафиксировано, агенту не выбирать заново)

1. **Персистентность состояния**: список торрентов + их id/infoHash хранить в sqlite (или json-файле для MVP) в отдельном volume. При рестарте контейнера — читать список и заново добавлять в anacrolix client; библиотека сама перепроверит уже скачанные куски по хешам на диске.
2. **Auth**: простой shared-secret через заголовок `X-API-Key`, без JWT/сессий — инструмент для одного пользователя. Frontend один раз спрашивает ключ и держит в localStorage (это реальное веб-приложение в браузере пользователя, не sandboxed Claude-артефакт, так что localStorage тут уместен).
3. **Volume для медиа**: локальный docker volume (не R2) — Jellyfin нужен быстрый файловый доступ, монтировать S3 излишне для MVP.
4. **Прогресс на фронте**: SSE, не polling — меньше нагрузки, мгновенный отклик.
5. **Jellyfin API key**: генерируется вручную один раз в админке Jellyfin, кладётся в env torrent-service.

## Фазы для агента (каждая — самостоятельно тестируемый инкремент)

**M1. Go-скелет API**
Роуты из контракта выше, in-memory storage (без anacrolix). Цель — зафиксировать контракт и проверить curl'ом.

**M2. Реальное скачивание**
Подключить `anacrolix/torrent`. `POST /torrents` реально добавляет магнит/файл, `GET` отдаёт реальный прогресс.

**M3. Персистентность + resume после рестарта**
sqlite/json store, чтение списка при старте контейнера.

**M4. SSE эндпоинт**
Живой поток статусов вместо ручного polling.

**M5. Jellyfin hook**
При переходе статуса в `completed` — вызов `/Library/Refresh`.

**M6. Docker Compose**
Три сервиса (torrent-service, jellyfin, frontend) + shared volume + healthchecks. Env через `.env`.

**M7. Frontend: добавление торрента**
Форма: paste magnet / drag-drop .torrent файла. Экран ввода API-ключа при первом заходе.

**M8. Frontend: список + живой прогресс**
Карточки/таблица с прогресс-баром, скоростью, статусом, кнопкой удаления. Подписка на SSE.

**M9. Деплой и e2e-проверка**
Завести за существующим Nginx + Cloudflare Tunnel на MainRenServer. Проверка целиком: кинул magnet с телефона → фильм появился на ТВ в Jellyfin.

## Открытые вопросы (реши до старта агента)

- Удалять ли файлы с диска при `DELETE`, если торрент ещё не докачан, — по умолчанию или спрашивать?
- Нужен ли лимит на одновременные закачки (anacrolix тянет несколько параллельно без проблем, но диск/канал могут быть узким местом)?
- Сидить после докачки или сразу останавливать раздачу (влияет на трафик/место)?