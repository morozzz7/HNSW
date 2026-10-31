# HNSW Coursework

Design of the HNSW data structure and its use for approximate nearest neighbor search.

## Status

- **Backend: implemented.** Datasets, index build, k-NN search, vector insert, statistics, SSE progress.
- **Frontend, `GET /graph`, `POST /benchmark`: planned, not implemented yet.** The repository currently contains
  only `frontend/.gitkeep`; those routes return `404 NOT_FOUND`.

## Stack

- **Backend:** Go 1.22+, module `hnsw-coursework`, HNSW from scratch (no third-party ANN libraries), stdlib `net/http`.
- **Frontend (planned):** React + TypeScript (Vite).
- **Vectors:** `[]float32`; internal IDs: `uint32`; distance: squared L2 internally (`sqrt` only in API output).

## Repository layout

```
backend/          Go backend: cmd/server, internal/{hnsw,bruteforce,dataset,projection,benchmark,api}
frontend/         reserved for the React + TypeScript (Vite) UI
docs/             PLAN.md, API.md
scripts/          dataset download scripts (datasets are not committed)
.github/          GitHub Actions CI
```

## Run

Backend (listens on `:8080`; override with `PORT`):

```bash
cd backend
make run        # go run ./cmd/server
make test       # go test ./...
make test-race  # go test -race ./...
make bench      # Go benchmarks
make vet        # go vet ./...
```

`PORT=9000 make run` starts the server on port 9000. From the repository root the same targets work as
`make -C backend test`.

Frontend (planned; not runnable yet):

```bash
cd frontend
npm install
npm run dev
```

API contract: `docs/API.md`. Plan and progress: `docs/PLAN.md`.

## Notes

- Large datasets (SIFT etc.) are never committed; download via `scripts/download_sift.sh`. HTTP upload accepts
  `.csv` and `.json` only; `.fvecs`/`.ivecs` are read by the Go loaders and benchmarks.
- CORS is enabled for `http://localhost:5173`.
- The in-memory dataset store has a 1 GiB budget; exceeding it returns `507 STORE_FULL`.
