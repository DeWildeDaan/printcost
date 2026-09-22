# --- Stage 1: build ---
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Pure-Go SQLite driver (modernc.org/sqlite) — no CGO/gcc needed.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/printcost ./cmd/printcost

# --- Stage 2: runtime ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build /out/printcost /usr/local/bin/printcost

ENV DB_PATH=/data/printcost.db
ENV PORT=8080
VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/printcost"]
