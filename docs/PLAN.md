# Coursework Project Plan

**Topic:** Design of the HNSW data structure and its use for approximate nearest neighbor search.

**Goal:** implement HNSW for approximate nearest neighbor search. The app must let the user upload vectors, build an index, run k-NN queries, and visualize the connection graph.

**Stack:** backend: Go (HNSW from scratch); frontend: React + TypeScript (Vite).

> **For agents:** this file is the single source of truth on progress. Read it fully before starting. After finishing a task, mark it `[x]` and add an entry to the "Change log" (bottom). Never redo `[x]` items. If blocked, mark `[~]` and describe the reason in "Blockers".
>
> **Reports:** after every reasonably global change (finished stage or task group, new module/page/endpoint, architectural decision, major refactoring, benchmark run) the agent writes a report in `docs/reports/` from `docs/reports/TEMPLATE.md` **in Russian**, and links it in the "Change log" and "Reports index". Criteria and rules are in `.opencode/agent/*.md`.
>
> **Language:** all files, code and comments in English; UI text and reports in Russian.

Legend: `[ ]` not started · `[~]` in progress / blocked · `[x]` done

**Current status:** Stage 0 (backend scope), Stage 1 (HNSW core), Stage 2 (Data) and Stage 3 (Brute force and quality metrics) complete. Stage 0.3 (frontend scaffolding) still in the frontend agent's domain.

---

## Mapping to coursework chapters

| Chapter | Content | Plan stages |
|---|---|---|
| Chapter 1 | Multilayer structure of HNSW; parameters `M`, `efConstruction`, `efSearch` | Stages 1-3 (core), theory |
| Chapter 2 | Visualization of the connection graph on different layers | Stages 5, 7 |
| Chapter 3 | Accuracy and speed comparison with brute force | Stages 4, 8 |

---

## Stage 0. Repository setup

- [x] 0.1 Structure: `backend/`, `frontend/`, `docs/`, `scripts/`, `.opencode/agent/`
- [x] 0.2 `backend/`: `go mod init hnsw-coursework`, `Makefile` (run, test, bench, lint)
- [~] 0.3 `frontend/`: Vite + React + TS (strict), Tailwind, ESLint, Prettier, Vitest
- [x] 0.4 `.gitignore` (binaries, `node_modules`, datasets, `*.idx`)
- [x] 0.5 `README.md` with run instructions (backend :8080, frontend :5173)
- [x] 0.6 `docs/API.md`: first version of the contract (see backend agent)
- [x] 0.7 `docs/reports/` exists, `TEMPLATE.md` in place

## Stage 1. HNSW core (backend, `internal/hnsw`): Chapter 1

- [x] 1.1 Distance metrics: L2 (squared), cosine, inner product + tests
- [x] 1.2 Typed min/max heaps by distance + tests
- [x] 1.3 Visited set (generation counter, `sync.Pool`)
- [x] 1.4 `Index` struct, `Config` (`M`, `Mmax0`, `EfConstruction`, `Metric`, `Seed`, `UseHeuristic`)
- [x] 1.5 Level generation: `floor(-ln(U)*mL)`, `mL = 1/ln(M)`, seed determinism
- [x] 1.6 `SEARCH-LAYER` (Algorithm 2)
- [x] 1.7 `SELECT-NEIGHBORS-SIMPLE` (Alg. 3) and `SELECT-NEIGHBORS-HEURISTIC` (Alg. 4)
- [x] 1.8 `INSERT` (Alg. 1): layer descent, bidirectional links, shrinking overflowing neighbors
- [x] 1.9 `K-NN-SEARCH` (Alg. 5) with per-query `efSearch`
- [x] 1.10 Thread safety (RWMutex), `go test -race` passes
- [x] 1.11 Graph invariant tests (degree <= Mmax, entry point on `maxLevel`, no self-loops)
- [x] 1.12 Quality test: recall@10 >= 0.95 on 5-10k random vectors
- [x] 1.13 Go benchmarks for Insert / Search

## Stage 2. Data (backend, `internal/dataset`, `internal/projection`)

- [x] 2.1 CSV loader
- [x] 2.2 JSON loader
- [x] 2.3 `.fvecs` / `.ivecs` loader (SIFT/GIST format)
- [x] 2.4 Generator: uniform / gaussian / clusters (n, dim, seed)
- [x] 2.5 Validation (consistent dimension, NaN/Inf, size limits)
- [x] 2.6 In-memory dataset store (+ metadata, preview)
- [x] 2.7 PCA -> 2D for visualization (cached after index build)
- [x] 2.8 `scripts/download_sift.sh`: download a SIFT subset (do not commit)

## Stage 3. Brute force and quality metrics (backend)

- [x] 3.1 `internal/bruteforce`: exact k-NN (linear scan, optionally parallel)
- [x] 3.2 recall@k against brute force
- [x] 3.3 Latency (p50/p95/p99), QPS, build time, memory estimate

## Stage 4. Benchmark module (backend, `internal/benchmark`): Chapter 3

- [ ] 4.1 Parameter grid run `M x efConstruction x efSearch`
- [ ] 4.2 Results: recall@k, QPS, latency, build time, speedup vs brute force
- [ ] 4.3 Heuristic vs simple neighbor selection comparison
- [ ] 4.4 Metric comparison (L2 / cosine), optional
- [ ] 4.5 Scaling: query time vs N (1k, 10k, 100k, 1M)
- [ ] 4.6 Export results to JSON/CSV (for tables in the written report)

## Stage 5. HTTP API (backend, `internal/api`)

- [ ] 5.1 Server skeleton, routing, CORS, unified errors, graceful shutdown
- [ ] 5.2 `POST/GET /datasets` (multipart upload, generation)
- [ ] 5.3 `POST /index`, `GET /index/status`, `GET /index/events` (SSE), `GET /index/stats`
- [ ] 5.4 `POST /vectors` (insert into a ready index)
- [ ] 5.5 `POST /search` (mode: hnsw / bruteforce, `trace`)
- [ ] 5.6 `SearchWithTrace` in core: per-layer path (visited nodes, closest per layer)
- [ ] 5.7 `GET /graph` (`level`, `limit`, `around`, PCA coordinates)
- [ ] 5.8 `POST /benchmark` (run + fetch results)
- [ ] 5.9 Save/load index to disk (optional)
- [ ] 5.10 Update `docs/API.md`

## Stage 6. Frontend: skeleton, Dataset and Index pages

- [ ] 6.1 Skeleton: routing, layout, navigation, theme
- [ ] 6.2 API client, zod schemas, TanStack Query, error handling
- [ ] 6.3 MSW mocks (`VITE_USE_MOCKS`) for backend-less development
- [ ] 6.4 DatasetPage: drag&drop upload, generation, preview
- [ ] 6.5 IndexPage: form `M`, `efConstruction`, metric, heuristic, seed + tooltips
- [ ] 6.6 IndexPage: build progress (SSE), stats, degree histogram

## Stage 7. Frontend: search and graph visualization: Chapter 2

- [ ] 7.1 SearchPage: query input, `k`, `efSearch`, mode HNSW / brute-force / both
- [ ] 7.2 Results table + HNSW vs brute-force comparison (intersection, recall, time)
- [ ] 7.3 GraphCanvas (Sigma.js + graphology): vertices and edges of the selected layer
- [ ] 7.4 LayerSelector: switch layers `0..maxLevel`, node/edge counters
- [ ] 7.5 Layout: PCA coordinates <-> force-directed
- [ ] 7.6 NodeInfoPanel: node level, neighbors per layer, vector
- [ ] 7.7 "Node neighborhood" mode (`around`, hop radius) and displayed-node limit
- [ ] 7.8 SearchPathOverlay: step-by-step layer descent animation (back/forward/play)
- [ ] 7.9 "Multilayer" mode (stacked layers, vertical links)
- [ ] 7.10 Legend, layer colors, entry point highlight

## Stage 8. Frontend: benchmark: Chapter 3

- [ ] 8.1 BenchmarkPage: parameter grid form, run, progress indicator
- [ ] 8.2 Chart recall@k vs efSearch
- [ ] 8.3 Chart QPS vs recall (Pareto curve)
- [ ] 8.4 Chart latency p50/p95/p99
- [ ] 8.5 Chart build time vs `M` / `efConstruction`
- [ ] 8.6 Comparison with brute force (speedup xN)
- [ ] 8.7 Results table + CSV and PNG export

## Stage 9. Integration and polish

- [ ] 9.1 End-to-end scenario: upload -> index -> search -> graph -> benchmark
- [ ] 9.2 Edge cases (empty dataset, k > N, wrong dimension, rebuild)
- [ ] 9.3 UI performance on 2000+ node graphs
- [ ] 9.4 `docker-compose.yml` (optional)
- [ ] 9.5 README: run instructions, screenshots

## Stage 10. Experiments and coursework text

- [ ] 10.1 Experiment datasets: random, clusters, SIFT subset (100k)
- [ ] 10.2 Run experiments, store raw results in `docs/results/`
- [ ] 10.3 Chapter 1: ANN overview, HNSW description, effect of `M`, `efConstruction`, `efSearch`
- [ ] 10.4 Chapter 2: implementation and visualization description (screenshots of layers and search path)
- [ ] 10.5 Chapter 3: tables and charts, conclusions on accuracy and speed vs brute force
- [ ] 10.6 Introduction, conclusion, references (Malkov & Yashunin, 2016, etc.)

---

## Questions for backend (filled by the frontend agent)

_None yet._

## Blockers

- 0.3: Vite + React + TS scaffolding — frontend agent's domain; backend agent must not touch `frontend/` (see `.opencode/agent/backend.md`).

## Decisions

- HNSW is implemented from scratch, no third-party ANN libraries.
- Vectors are `float32`; internal IDs are `uint32`.
- `efSearch` is passed per query, not stored in the index.
- `hnsw.Search` / `SearchWithTrace` return exported `[]hnsw.Result` (`ID`, `Dist`) so results are usable outside the package (bruteforce comparison, HTTP API).
- Visualization coordinates (PCA -> 2D) are computed by the backend.
- API contract source of truth: `docs/API.md`.

## Reports index

| # | Date | Agent | Report | Stage / tasks |
|---|---|---|---|---|
| 1 | 2026-10-26 | backend | `docs/reports/2026-10-26_01_backend_etape-0-nastroika-repozytoriya.md` | Этап 0 (0.1, 0.2, 0.4–0.7) |
| 2 | 2026-10-26 | backend | `docs/reports/2026-10-26_02_backend_etap-1-yadro-hnsw.md` | Этап 1 (1.1–1.13) |
| 3 | 2026-10-27 | backend | `docs/reports/2026-10-27_03_backend_etap-2-sloj-dannyh.md` | Этап 2 (2.1–2.8) |
| 4 | 2026-10-27 | backend | `docs/reports/2026-10-27_04_backend_etap-3_bruteforce-metrics.md` | Этап 3 (3.1–3.3) |

## Change log

| Date | Agent | What was done |
|---|---|---|
| 2026-10-26 | backend | Этап 1 (1.1–1.13): реализовано ядро HNSW с нуля в `backend/internal/hnsw` — метрики расстояний (L², cosine, inner product), типизированные кучами min/max, посещение с generation counter + `sync.Pool`, `Index`/`Config` с генерацией уровня `floor(-ln(U)·mL)`, SEARCH-LAYER (Алг. 2), SELECT-NEIGHBORS (Алг. 3 и 4), INSERT (Алг. 1) с двунаправленными ссылками и обрезкой переполненных соседей, K-NN-SEARCH (Алг. 5) с per-query `efSearch`, `SearchWithTrace`, `Stats`. Тесты: юнит-тесты всех компонентов, инварианты графа, детерминизм по seed, recall@10 = 1.0000 на 10k случайных векторов (M=16, efC=200, efSearch=100, dim=16, 100 запросов), `go test -race` пройден (102.8s). Бенчмарки: Insert ~87.7 мс на 1000 вставок, Search ~67.9 мкс/запрос. Зависимости не добавлялись (алгоритм реализован с нуля). Отчёт: `docs/reports/2026-10-26_02_backend_etap-1-yadro-hnsw.md`. |
| 2026-10-27 | backend | Этап 2 (2.1–2.8): слой данных в `backend/internal/dataset` и `backend/internal/projection` — загрузчики CSV/JSON/.fvecs/.ivecs, генераторы uniform/gaussian/clusters, валидация (размерность, NaN/Inf, лимиты), in-memory `Store` с метаданными и preview, PCA -> 2D с кэшем, скрипт `scripts/download_sift.sh`. Тесты: unit-тесты загрузчиков, генератора, Store, валидации и PCA. `go vet`/`go test`/`gofmt` чисто. Отчёт: `docs/reports/2026-10-27_03_backend_etap-2-sloj-dannyh.md`. |
| 2026-10-26 | backend | Этап 0: структура репозитория (`backend/`, `frontend/`, `scripts/`), `backend/go.mod` (`hnsw-coursework`), `backend/Makefile` (run/test/bench/vet/lint), корневой `.gitignore`, `README.md`, `docs/API.md` (version 1 контракта). `go vet`/`go test` пока без пакетов (кода нет) — проверка станет обязательна с Этапа 1. Отчёт: `docs/reports/2026-10-26_01_backend_etape-0-nastroika-repozytoriya.md`. |
| 2026-10-27 | backend | Этап 3 (3.1–3.3): `backend/internal/bruteforce` — точный k-NN линейным сканом (последовательно и параллельно через `NewWithWorkers`; результат детерминирован, тай-брейк по id), `Recall` (recall@k по множеству id); `backend/internal/benchmark` — перцентили задержки p50/p95/p99 (линейная интерполяция), QPS по средней задержке, `MeasureBuildTime`, оценка памяти через `HeapInUse`; интеграционный тест качества HNSW vs bruteforce. Результаты: recall@10 = 1.0000 (n=3000, dim=16, M=16, efC=100, efSearch=50, 30 запросов), построение 165.6 мс, память 824 KiB, p50=0.033 мс, p95=0.052 мс, p99=0.066 мс, QPS ≈ 27 362. Бенчмарки: bruteforce 0.965 мс/запрос (n=10000, dim=16, k=10) ≈ 1036 QPS, параллельный (4 воркера) 0.316 мс ≈ 3163 QPS (ускорение ≈3.05×); HNSW 61.8 мкс/запрос (n=5000, efSearch=100). `hnsw.Search`/`SearchWithTrace` теперь возвращают экспортируемый `[]hnsw.Result` (`ID`, `Dist`) вместо внутреннего `item` — для сравнения из других пакетов и будущего API (поведение не менялось). `go vet`, `gofmt`, `go test`, `go test -race` — чисто. Отчёт: `docs/reports/2026-10-27_04_backend_etap-3_bruteforce-metrics.md`. |
