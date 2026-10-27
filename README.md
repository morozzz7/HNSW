# HNSW Coursework

Design of the HNSW data structure and its use for approximate nearest neighbor search.

## Stack

- **Backend:** Go 1.22+, module `hnsw-coursework`, HNSW from scratch (no third-party ANN libraries), stdlib `net/http`.
- **Frontend:** React + TypeScript (Vite).
- **Vectors:** `[]float32`; internal IDs: `uint32`; distance: squared L2 (sqrt only in API output).

## Repository layout

```
backend/          Go backend: cmd/server, internal/{hnsw,bruteforce,dataset,projection,benchmark,api}
frontend/         React + TypeScript (Vite) UI
docs/             PLAN.md, API.md, reports/, results/
scripts/          dataset download scripts (datasets are not committed)
```

## Run

Backend (listens on `:8080`):

```bash
cd backend
make run        # go run ./cmd/server
make test       # go test ./...
make test-race  # go test -race ./...
make bench      # Go benchmarks
make vet        # go vet ./...
```

Frontend (runs on `:5173`):

```bash
cd frontend
npm install
npm run dev     # Vite dev server
```

API contract: `docs/API.md`. Plan and progress: `docs/PLAN.md`.

## Notes

- Large datasets (SIFT etc.) are never committed; download via `scripts/download_sift.sh`.
- CORS is enabled for `http://localhost:5173`.
