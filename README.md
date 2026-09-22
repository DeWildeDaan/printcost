# 3D Print Cost Calculator

Self-hosted app for calculating the cost of 3D prints, including filament,
machine time/depreciation, consumables, AMS/nozzle wear, and other equipment
costs. Single Go binary with an embedded static frontend and a SQLite
database — no external dependencies to run.

## Stack

- Go ([go-chi](https://github.com/go-chi/chi) router, [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) pure-Go driver — no CGO)
- Plain HTML/CSS/JS frontend, embedded into the binary at build time
- SQLite for storage

## Running locally

```
go run ./cmd/printcost
```

The app listens on `:8080` and creates `./printcost.db` on first run.

Configuration is via environment variables:

| Variable          | Default            | Description                          |
|-------------------|---------------------|---------------------------------------|
| `PORT`             | `8080`              | HTTP listen port                     |
| `DB_PATH`          | `./printcost.db`    | Path to the SQLite database file     |
| `BASE_URL_PREFIX`  | *(empty)*           | Prefix to serve the app under, e.g. `/printcost` |

## Tests

```
go test ./...
```

## Docker

```
docker build -t printcost .
docker run -p 8080:8080 -v printcost-data:/data printcost
```

## Helm

A Helm chart is provided in [`helm/printcost`](helm/printcost) for deploying
to Kubernetes. See [`helm/printcost/README.md`](helm/printcost/README.md) for
details — note that only a single replica is supported (SQLite is
single-writer).

```
helm install my-printcost ./helm/printcost \
  --set image.repository=your-registry/printcost \
  --set image.tag=1.0.0
```

## CI/CD

On every push to `main` and on version tags (`v*`), GitHub Actions:

- runs `go test ./...`
- builds and pushes the Docker image to `ghcr.io/<owner>/3d-print-cost-calculator`
- packages the Helm chart and pushes it as an OCI artifact to the same GHCR namespace

See [`.github/workflows/ci.yml`](.github/workflows/ci.yml).
