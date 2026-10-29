# HTTP API Contract (v1)

Source of truth for the backend (`backend/internal/api`) and the frontend.
All files and code comments are in English; this contract is part of the repository documentation.

**Base path:** `/api/v1`
**Content-Type:** `application/json` for all endpoints except `POST /datasets` (`multipart/form-data`).
**CORS:** enabled for `http://localhost:5173` (preflight `OPTIONS` returns `204`).
**Vectors:** `[]float32`. Distances are compared as squared L2; `sqrt` is applied only in API output.
**IDs:** internal graph IDs are `uint32`; external IDs are provided by the client or assigned by the server.

The implementation also exposes `GET /health` (`200 {"status":"ok"}`); it is not part of the documented
application flow and is only used for liveness checks.

---

## Error format

Every error response (non-2xx) has the body:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "human readable message"
  }
}
```

Standard error codes:

| Code | HTTP | Meaning |
|---|---|---|
| `INVALID_REQUEST` | 400 | malformed body, missing or invalid parameter |
| `DATASET_NOT_FOUND` | 404 | dataset id unknown |
| `INDEX_NOT_FOUND` | 404 | no index built yet |
| `INDEX_NOT_READY` | 409 | index is still building or in error state |
| `INVALID_DIMENSION` | 400 | vector dimension does not match dataset/index |
| `K_EXCEEDS_N` | 400 | requested `k` is larger than the number of indexed vectors |
| `BUILD_FAILED` | 500 | index build failed (see message) |
| `SEARCH_FAILED` | 500 | search failed (see message) |
| `FILE_TOO_LARGE` | 413 | dataset upload exceeds the 256 MiB limit |
| `UNSUPPORTED_FILE_TYPE` | 415 | upload is not a `.csv` or `.json` file |
| `NOT_FOUND` | 404 | unknown route |
| `METHOD_NOT_ALLOWED` | 405 | known route, wrong HTTP method |
| `INTERNAL_ERROR` | 500 | unexpected server error |

Unknown paths and wrong methods also use the envelope above (`NOT_FOUND` / `METHOD_NOT_ALLOWED`).

---

## Endpoints

### 1. Datasets

#### `POST /datasets`

Upload a dataset file (CSV or JSON) or generate a synthetic one.

**Multipart upload:** form field `file` with a `.csv` or `.json` file (maximum 256 MiB; other extensions are
rejected with `UNSUPPORTED_FILE_TYPE`).

**Generation (JSON body):**

```json
{
  "action": "generate",
  "n": 1000,
  "dim": 16,
  "distribution": "uniform",   // uniform | gaussian | clusters
  "seed": 1
}
```

**Response (201):**

```json
{
  "id": "ds_53a1f2",
  "n": 1000,
  "dim": 16,
  "preview": [[0.12, -0.05, ...], ...]   // first few vectors
}
```

#### `GET /datasets`

List all uploaded/generated datasets.

**Response (200):**

```json
{
  "datasets": [
    { "id": "ds_53a1f2", "n": 1000, "dim": 16, "createdAt": "2026-10-26T19:00:00Z" }
  ]
}
```

#### `GET /datasets/{id}`

Dataset metadata and preview.

**Response (200):**

```json
{
  "id": "ds_53a1f2",
  "n": 1000,
  "dim": 16,
  "preview": [[...], ...],
  "stats": { "min": [...], "max": [...], "mean": [...] },
  "createdAt": "2026-10-26T19:00:00Z",
  "source": "generate",              // csv | json | generate
  "distribution": "uniform"          // present for generated datasets only
}
```

The `GET /datasets/{id}` response may include `createdAt`, `source` and `distribution` in addition to the
fields above. Generation is capped at `n * dim <= 50_000_000` float components.

---

### 2. Index

#### `POST /index`

Build an HNSW index over a dataset. Returns immediately; build runs asynchronously.

**Request:**

```json
{
  "datasetId": "ds_53a1f2",
  "M": 16,
  "efConstruction": 200,
  "metric": "l2",          // l2 | cosine | inner_product
  "heuristic": true,
  "seed": 42
}
```

**Defaults for omitted fields:** `M=16`, `efConstruction=200`, `metric="l2"`, `heuristic=true`, `seed=0`.

**Build semantics:** the build runs asynchronously. While it runs, `GET /index/status` reports `building`
and search/insert/stats requests return `INDEX_NOT_READY`. A second build request while a build is running
is rejected with `409 INDEX_NOT_READY`. Rebuilding when an index is already `ready` is allowed: the previous
index is discarded, the new one is built from scratch and searches are unavailable until it is ready.

**Response (202):**

```json
{
  "indexId": "idx_7b1c9d",
  "status": "building"
}
```

#### `GET /index/status`

Build status: `idle | building | ready | error`.

**Response (200):**

```json
{
  "status": "building",
  "progress": 0.37,           // 0..1
  "elapsedMs": 1240,
  "indexId": "idx_7b1c9d",
  "error": null
}
```

Before the first build `status` is `idle`, `progress` is `0`, `indexId` is `""` and `error` is `null`.
`error` is non-null only when `status` is `error`.

#### `GET /index/events`

SSE stream with build progress. Events: `progress` (payload: `{ progress, processed, total }`), `done`, `error`.

```
event: progress
data: {"progress":0.37,"processed":3700,"total":10000}

event: done
data: {"indexId":"idx_7b1c9d"}
```

On connect the server sends a snapshot of the current state (a `progress` event while building, a `done`
or `error` event followed by closing the stream when the build is already finished, or a comment while
idle). Progress events are throttled to about 100 per build. Comment heartbeats (`: ping`) keep idle
connections alive; the stream closes after `done`/`error` or when the client disconnects.

#### `GET /index/stats`

Index statistics after build.

**Response (200):**

```json
{
  "nodes": 10000,
  "maxLevel": 5,
  "entryPoint": 891,
  "nodesPerLevel": [10000, 3983, 1462, 537, 197, 72, 3],
  "degreeDistribution": { "min": 2, "max": 32, "mean": 16.4, "p50": 16 },
  "metric": "l2",
  "M": 16,
  "efConstruction": 200
}
```

`degreeDistribution` is computed over every `(node, level)` pair present in the graph (level 0 dominates
for typical `M`). The response may also include `indexId` and `datasetId`. `GET /index/stats` returns
`INDEX_NOT_FOUND` (404) before the first build, `INDEX_NOT_READY` (409) while building, and `BUILD_FAILED`
(500) if the build failed.

---

### 3. Vectors

#### `POST /vectors`

Insert a single vector into a ready index.

**Request:**

```json
{
  "vectorId": "ext_123",      // optional; server-assigned if omitted
  "vector": [0.12, -0.05, ...]
}
```

**Response (201):**

```json
{
  "id": 42,
  "vectorId": "ext_123"
}
```

Dataset vectors use their row index as the external id (`"0"`, `"1"`, ...). When `vectorId` is omitted,
the server assigns `"ext_<n>"`. A duplicate external id returns `INVALID_REQUEST` (400), a dimension
mismatch returns `INVALID_DIMENSION` (400), and NaN/Inf components are rejected with `INVALID_REQUEST`.
The inserted vector is immediately added to both the HNSW index and the brute-force counterpart.

---

### 4. Search

#### `POST /search`

k-NN search. `efSearch` is passed per query; `ef = max(efSearch, k)` internally.

**Request:**

```json
{
  "vector": [0.12, -0.05, ...],   // or "vectorId": 42
  "k": 10,
  "efSearch": 100,
  "mode": "hnsw",                 // hnsw | bruteforce
  "trace": false                  // if true, include per-layer search path
}
```

**Response (200):**

```json
{
  "results": [
    { "id": 42, "vectorId": "ext_123", "distance": 1.44, "vector": [...] }
  ],
  "found": 10,
  "trace": {                        // only when "trace": true
    "layers": [
      { "level": 5, "visited": [891, 72, 3], "closest": 3 },
      { "level": 0, "visited": [3, 12, 45, ...], "closest": 42 }
    ]
  }
}
```

Distances in output have `sqrt` applied (actual L2 / cosine distance, not squared).

**Request rules:**

- Provide exactly one of `vector` or `vectorId`. `vectorId` may be a string or a number; numbers are
  compared against the decimal external id (`0` resolves the dataset row `"0"`).
- `mode` defaults to `"hnsw"`. `efSearch` is optional: `ef = max(efSearch, k)`; when omitted or `0`
  it becomes `ef = k`.
- `trace` is honoured only for `mode="hnsw"`; brute-force responses never contain a `trace` field.
- Errors: unknown `vectorId` → `INVALID_REQUEST` (400), `k > N` → `K_EXCEEDS_N` (400), wrong dimension →
  `INVALID_DIMENSION` (400), no index yet → `INDEX_NOT_FOUND` (404), building/error index →
  `INDEX_NOT_READY` (409).

**Distance representation by metric:** `l2` — actual Euclidean distance (`sqrt` of the squared L2 used
internally); `cosine` — `1 - cosine similarity`; `inner_product` — negative dot product, so smaller
values mean closer vectors (same convention as the internal index).

---

### 5. Graph (visualization)

#### `GET /graph`

Vertices and edges of a layer, with a vertex-count limit. PCA 2D coordinates are attached.

**Query params:** `level` (default 0), `limit` (max vertices, default 500), `around` (optional external/internal id to center the view on).

**Response (200):**

```json
{
  "level": 0,
  "limit": 500,
  "vertices": [
    { "id": 42, "vectorId": "ext_123", "level": 5, "x": 0.12, "y": -0.5, "isEntryPoint": false }
  ],
  "edges": [
    { "from": 42, "to": 17, "level": 0 }
  ],
  "totalVertices": 10000
}
```

---

### 6. Benchmark

#### `POST /benchmark`

Run HNSW vs brute-force comparison over a parameter grid.

**Request:**

```json
{
  "indexId": "idx_7b1c9d",
  "M": [8, 16, 32],
  "efConstruction": [100, 200],
  "efSearch": [10, 50, 100],
  "k": 10,
  "queries": 100
}
```

**Response (200):**

```json
{
  "id": "bench_3f9a",
  "results": [
    {
      "M": 16,
      "efConstruction": 200,
      "efSearch": 100,
      "recall@10": 0.968,
      "qps": 3420.5,
      "latencyP50Ms": 0.12,
      "latencyP95Ms": 0.31,
      "latencyP99Ms": 0.58,
      "buildTimeMs": 412,
      "bruteforceQps": 12.4,
      "speedup": 275.8
    }
  ]
}
```

`POST /benchmark` is async in production; use `GET /benchmark/{id}` to poll (contract v1 returns inline for small grids).
