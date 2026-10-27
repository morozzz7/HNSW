# HTTP API Contract (v1)

Source of truth for the backend (`backend/internal/api`) and the frontend.
All files and code comments are in English; this contract is part of the repository documentation.

**Base path:** `/api/v1`
**Content-Type:** `application/json` for all endpoints except `POST /datasets` (`multipart/form-data`).
**CORS:** enabled for `http://localhost:5173`.
**Vectors:** `[]float32`. Distances are compared as squared L2; `sqrt` is applied only in API output.
**IDs:** internal graph IDs are `uint32`; external IDs are provided by the client or assigned by the server.

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

---

## Endpoints

### 1. Datasets

#### `POST /datasets`

Upload a dataset file (CSV or JSON) or generate a synthetic one.

**Multipart upload:** form field `file` with a `.csv` or `.json` file.

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
  "stats": { "min": [...], "max": [...], "mean": [...] }
}
```

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

#### `GET /index/events`

SSE stream with build progress. Events: `progress` (payload: `{ progress, processed, total }`), `done`, `error`.

```
event: progress
data: {"progress":0.37,"processed":3700,"total":10000}

event: done
data: {"indexId":"idx_7b1c9d"}
```

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
