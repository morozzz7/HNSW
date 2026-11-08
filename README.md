# HNSW Coursework

Design of the HNSW data structure and its use for approximate nearest neighbor search.

## Status

- **Backend: implemented.** Datasets, index build, k-NN search, vector insert, statistics, SSE progress,
  graph snapshot, benchmark jobs, save/load.
- **Frontend: in progress.** Skeleton, API layer, dataset and index screens are in place (stage 6.4–6.6);
  search and benchmark screens follow.

## Stack

- **Backend:** Go 1.22+, module `hnsw-coursework`, HNSW from scratch (no third-party ANN libraries), stdlib `net/http`.
- **Frontend:** React 19 + TypeScript (Vite 8), Tailwind CSS 4, TanStack Query, zod, MSW, Vitest.
- **Vectors:** `[]float32`; internal IDs: `uint32`; distance: squared L2 internally (`sqrt` only in API output).

## Repository layout

```
backend/          Go backend: cmd/server, internal/{hnsw,bruteforce,dataset,projection,benchmark,api}
frontend/         React + TypeScript (Vite) UI
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

Frontend (listens on `:5173` and proxies `/api` to `http://localhost:8080`):

```bash
cd frontend
npm install
npm run dev          # dev server, real backend
npm run dev:mocks    # dev server with MSW mocks (no backend needed)
npm run build        # type-check + production build
npm run lint         # ESLint
npm run test         # Vitest
```

API contract: `docs/API.md`. Plan and progress: `docs/PLAN.md`.

## Notes

- Large datasets (SIFT etc.) are never committed; download via `scripts/download_sift.sh`. HTTP upload accepts
  `.csv` and `.json` only; `.fvecs`/`.ivecs` are read by the Go loaders and benchmarks.
- CORS is enabled for `http://localhost:5173`; the Vite dev server also proxies `/api` so the UI does not depend on CORS.
- The in-memory dataset store has a 1 GiB budget; exceeding it returns `507 STORE_FULL`.
