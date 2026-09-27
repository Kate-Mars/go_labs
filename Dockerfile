# syntax=docker/dockerfile:1.7

# --- build stage -----------------------------------------------------------
FROM golang:1.27-alpine AS build

WORKDIR /src

# Сначала только манифесты — так кешируется слой с зависимостями.
COPY go.mod go.sum ./
RUN go mod download

# Потом исходники.
COPY . .

# Собираем статический бинарь. CGO выключен — бинарь не тянет libc,
# значит можно положить его в scratch/distroless без пакетов.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/trip-service ./cmd/trip-service

# --- runtime stage ---------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

# nonroot-тег: пользователь uid=65532, у него нет прав root.
COPY --from=build /out/trip-service /trip-service

EXPOSE 8080

ENTRYPOINT ["/trip-service"]