# Использовать базовый образ Go
FROM golang:1.24-alpine AS builder

# Установка зависимостей сборки. ca-certificates нужен для работы с HTTPS.
RUN apk add --no-cache ca-certificates git

# Установка рабочей директории внутри контейнера
WORKDIR /app

# Копирование go.mod и go.sum и загрузка зависимостей
# Это позволяет кэшировать зависимости, если go.mod/go.sum не меняются
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Копирование ВСЕГО исходного кода приложения
# Теперь это включает в себя и cmd/server
COPY . .

# Сборка Go-приложения
# -o app: указывает имя выходного исполняемого файла
# ./cmd/server: указывает, что собирать нужно пакет в этой директории
# -ldflags: для уменьшения размера
RUN CGO_ENABLED=0 go build -o app -ldflags "-s -w" ./cmd/server

# --- Вторая стадия: создание легковесного образа для продакшна ---
FROM alpine:latest

# Убедитесь, что ca-certificates установлены и здесь (для HTTPS)
RUN apk add --no-cache ca-certificates

# Установка рабочей директории
WORKDIR /app

# Копирование исполняемого файла из предыдущей стадии
COPY --from=builder /app/app .

# Создание внутренней папки /app/data, если она не существует.
# Это папка, куда будет монтироваться внешний том data.
# Если вы не создадите ее, Docker создаст ее сам, но лучше быть явным.
RUN mkdir -p /app/data

# Определение порта, который приложение будет слушать
EXPOSE 8081

# Команда для запуска приложения
ENTRYPOINT ["./app"]