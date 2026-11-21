# HNSW Coursework

![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![TypeScript 6](https://img.shields.io/badge/TypeScript-6-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![React 19](https://img.shields.io/badge/React-19-61DAFB?style=for-the-badge&logo=react&logoColor=white)
![Vite 8](https://img.shields.io/badge/Vite-8-646CFF?style=for-the-badge&logo=vite&logoColor=white)
![Tailwind CSS 4](https://img.shields.io/badge/Tailwind-4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)

![TanStack Query 5](https://img.shields.io/badge/TanStack%20Query-5-FF4154?style=for-the-badge&logo=tanstack&logoColor=white)
![Sigma.js 3](https://img.shields.io/badge/Sigma.js-3-5B8FF9?style=for-the-badge)
![HNSW from scratch](https://img.shields.io/badge/HNSW-from%20scratch-5BE0B5?style=for-the-badge)
![CI passing](https://img.shields.io/badge/CI-passing-2EA043?style=for-the-badge&logo=githubactions&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)

**English** · [Русский](README_RU.md)

Design of the HNSW data structure and its use for approximate nearest neighbor search.

## Status

All planned stages are complete: HNSW core, dataset loaders and generators, brute-force reference and
quality metrics, benchmark module, HTTP API, CLI-verified backend hardening, and the React UI
(datasets, index build, search, graph visualization, benchmark). Stage 9 covers the end-to-end
scenario, edge cases, graph performance, Docker Compose and documentation.

## Architecture

### Service

```mermaid
flowchart LR
  subgraph Client["Browser · :5173"]
    Pages["React pages<br/>Datasets · Build · Search · Benchmark"]
    Hooks["TanStack Query<br/>zod validation"]
  end

  subgraph Edge["Vite dev server · :5173"]
    Proxy["/api → :8080<br/>VITE_API_PROXY_TARGET"]
    Mock["MSW mocks<br/>VITE_USE_MOCKS"]
  end

  subgraph HTTP["Go HTTP API · :8080"]
    Routes["net/http<br/>CORS · errors · SSE"]
    Manager["indexmanager<br/>build state · external IDs · RWMutex"]
  end

  subgraph Core["Core packages"]
    HNSW["hnsw<br/>graph + search"]
    Brute["bruteforce<br/>exact k-NN"]
    Data["dataset + projection<br/>loaders · PCA"]
    Bench["benchmark<br/>grid · metrics · export"]
  end

  subgraph Storage["Data"]
    Files["CSV / JSON / .fvecs"]
    Memory["In-memory store · 1 GiB"]
    Snapshot["Index snapshot · *.idx"]
  end

  Pages --> Hooks --> Proxy
  Mock -.-> Hooks
  Proxy --> Routes --> Manager
  Manager --> HNSW
  Manager --> Brute
  Manager --> Data
  Manager --> Bench
  Data --> Files
  Data --> Memory
  Manager --> Snapshot
  Bench --> HNSW
  Bench --> Brute
  Routes -- "SSE: progress / done / error" --> Pages

  click Pages "docs/screenshots/03-search-graph.png" "UI screenshot"
  click Proxy "frontend/vite.config.ts" "Vite proxy configuration"
  click Routes "docs/API.md" "HTTP contract"
  click Manager "backend/internal/indexmanager" "Index manager source"
  click HNSW "backend/internal/hnsw" "HNSW core source"
  click Brute "backend/internal/bruteforce" "Brute force source"
  click Data "backend/internal/dataset" "Dataset source"
  click Bench "backend/internal/benchmark" "Benchmark source"
```

### HNSW algorithm

```mermaid
flowchart TB
  subgraph Graph["Multilayer structure"]
    Entry["L2 · entry point"]
    Mid["L1 · sparse long links"]
    Base["L0 · all vectors"]
    Entry --> Mid --> Base
  end

  subgraph InsertFlow["INSERT — Algorithm 1"]
    I1["Level: floor(-ln(U) · mL)"]
    I2["SEARCH-LAYER from the entry point"]
    I3["SELECT-NEIGHBORS-HEURISTIC"]
    I4["Bidirectional links + shrink on overflow"]
    I1 --> I2 --> I3 --> I4
  end

  subgraph SearchFlow["K-NN SEARCH — Algorithm 5"]
    S1["Start at the entry point on maxLevel"]
    S2["Greedy descent: one closest node per layer"]
    S3["SEARCH-LAYER on L0 with ef = max(efSearch, k)"]
    S4["Return k nearest by distance"]
    S5["Recall@k against brute force"]
    S1 --> S2 --> S3 --> S4 --> S5
  end

  subgraph Primitives["Shared primitives"]
    Dist["squared L2 / cosine / inner product"]
    Visited["visited set: generation + sync.Pool"]
    Heap["typed min/max heaps"]
    Select["simple / heuristic neighbor selection"]
  end

  Entry --> I2
  Base --> I4
  Entry --> S1
  Base --> S3
  Dist --> S3
  Visited --> S2
  Heap --> S3
  Select --> I3

  click I1 "backend/internal/hnsw/index.go" "Level generation"
  click I2 "backend/internal/hnsw/insert.go" "INSERT and SEARCH-LAYER call sites"
  click I3 "backend/internal/hnsw/neighbors.go" "Neighbor selection algorithms"
  click S2 "backend/internal/hnsw/search.go" "SEARCH-LAYER and K-NN SEARCH"
  click Dist "backend/internal/hnsw/distance.go" "Distance metrics"
  click Visited "backend/internal/hnsw/visited.go" "Visited set"
  click Heap "backend/internal/hnsw/heap.go" "Typed heaps"
  click S5 "backend/internal/bruteforce" "Exact reference search"
```

## Stack

- **Backend:** Go 1.22+, module `hnsw-coursework`, HNSW from scratch (no third-party ANN libraries), stdlib `net/http`.
- **Frontend:** React 19 + TypeScript (Vite 8), Tailwind CSS 4, TanStack Query 5, React Router 7, zod, MSW, Vitest, Sigma.js v3 + Graphology.
- **Vectors:** `[]float32`; internal IDs: `uint32`; squared L2 internally (`sqrt` only in API output).

## Repository layout

```
README_RU.md       Russian version of this document
backend/            Go backend: cmd/server, internal/{hnsw,bruteforce,dataset,projection,benchmark,api,indexmanager}
frontend/           React + TypeScript (Vite) UI
docs/               PLAN.md, API.md, screenshots/
scripts/            dataset download scripts (datasets are not committed)
.github/workflows/  GitHub Actions CI
docker-compose.yml  local backend + frontend in containers
```

## Run

Backend (listens on `:8080`, override with `PORT`):

```bash
cd backend
make run        # go run ./cmd/server
```

Frontend (listens on `:5173`, proxies `/api` to `http://localhost:8080`):

```bash
cd frontend
npm install
npm run dev          # with a real backend
npm run dev:mocks    # with MSW mocks, no backend needed
npm run build        # type-check + production build
```

From the repository root the backend targets also work as `make -C backend <target>`.

Docker Compose (both services, frontend on :5173, backend on :8080):

```bash
docker-compose up --build
docker-compose down
```

In containers the Vite proxy target comes from `VITE_API_PROXY_TARGET` (default
`http://localhost:8080`); Compose sets it to `http://backend:8080`.

## Workflow

1. **Datasets** (`/datasets`) — upload `.csv`/`.json` via drag-and-drop or generate a synthetic set
   (uniform, gaussian, clusters) with a fixed seed; inspect preview and coordinate statistics.

   ![Dataset list and preview](docs/screenshots/01-datasets.png)

2. **Build** (`/index`) — choose a dataset, `M`, `efConstruction`, metric, neighbor-selection heuristic
   and seed. Build progress arrives over SSE; the finished screen shows node counts per level and the
   degree histogram.

   ![Index build and graph statistics](docs/screenshots/02-index.png)

3. **Search** (`/search`) — query by vector ID or by explicit coordinates, choose `k`, `efSearch` and
   HNSW / brute force / both. The results table reports Recall@k, overlap and speedup; the WebGL graph
   supports layer switching, PCA or ForceAtlas2 layout, node inspection, neighborhood mode and a
   step-by-step trace of the layer descent.

   ![Search results and graph visualization](docs/screenshots/03-search-graph.png)

4. **Benchmark** (`/benchmark`) — run an `M × efConstruction × efSearch` grid and inspect recall,
   Pareto QPS, latency percentiles, build time and the results table; charts export to PNG and the
   table to CSV.

   ![Benchmark charts and results](docs/screenshots/04-benchmark.png)

## Verification

```bash
cd backend && gofmt -l . && go vet ./... && go test ./... && go test -race ./...
cd frontend && npm run format:check && npm run lint && npm run test && npm run build
```

The full UI scenario was also exercised end-to-end in Chromium against the real backend:
generate dataset → build index → search (including `k > N` and rebuild) → graph → benchmark.
The graph was checked at 2000 nodes / ~17k edges with no console errors.

API contract: `docs/API.md`. Plan and progress: `docs/PLAN.md`.

## Notes

- Large datasets (SIFT etc.) are never committed; download via `scripts/download_sift.sh`. HTTP upload accepts
  `.csv` and `.json` only; `.fvecs`/`.ivecs` are read by the Go loaders and benchmarks.
- CORS is enabled for `http://localhost:5173`; the Vite dev server also proxies `/api` so the UI does not depend on CORS.
- The in-memory dataset store has a 1 GiB budget; exceeding it returns `507 STORE_FULL`.
