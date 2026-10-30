#!/usr/bin/env bash
set -euo pipefail

DATASET_DIR="${DATASET_DIR:-/tmp/opencode/datasets}"
mkdir -p "$DATASET_DIR"

N_BASE="${1:-50000}"
N_QUERIES="${2:-1000}"
DIM=128

BASE_URL="https://ftp.irisa.fr/local/texmex/corpus/sift.tgz"
BASE_TGZ="$DATASET_DIR/sift.tgz"
BASE_DIR="$DATASET_DIR/sift"

BASE_FVECS="$DATASET_DIR/sift_base_subset.fvecs"
QUERY_FVECS="$DATASET_DIR/sift_query_subset.fvecs"

log() { printf '[download_sift] %s\n' "$*"; }

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || {
        log "error: required command '$1' not found"
        exit 1
    }
}
need_cmd curl || need_cmd wget || true
if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    log "error: neither curl nor wget is available"
    exit 1
fi

download() {
    local url="$1" out="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 -o "$out" "$url"
    else
        wget -q -O "$out" "$url"
    fi
}

take_fvecs() {
    local src="$1" dst="$2" n="$3"
    python3 - "$src" "$dst" "$n" <<'PY'
import struct, sys
src, dst, n = sys.argv[1], sys.argv[2], int(sys.argv[3])
with open(src, "rb") as f, open(dst, "wb") as g:
    for _ in range(n):
        header = f.read(4)
        if len(header) < 4:
            break
        (dim,) = struct.unpack("<i", header)
        vec = f.read(4 * dim)
        if len(vec) < 4 * dim:
            break
        g.write(header)
        g.write(vec)
PY
}

main() {
    log "dataset dir: $DATASET_DIR"
    log "subset: $N_BASE base + $N_QUERIES queries (dim=$DIM)"

    if [ ! -f "$BASE_TGZ" ]; then
        log "downloading sift.tgz (~250 MB) from $BASE_URL ..."
        download "$BASE_URL" "$BASE_TGZ"
    else
        log "sift.tgz already present, skipping download"
    fi

    if [ ! -d "$BASE_DIR" ]; then
        log "extracting sift.tgz ..."
        tar -xzf "$BASE_TGZ" -C "$DATASET_DIR"
    else
        log "sift/ already extracted, skipping"
    fi

    BASE_SRC=$(find "$BASE_DIR" -name 'sift_base.fvecs' | head -n1)
    QUERY_SRC=$(find "$BASE_DIR" -name 'sift_query.fvecs' | head -n1)
    if [ -z "$BASE_SRC" ] || [ -z "$QUERY_SRC" ]; then
        log "error: sift_base.fvecs / sift_query.fvecs not found in $BASE_DIR"
        exit 1
    fi

    take_fvecs "$BASE_SRC" "$BASE_FVECS" "$N_BASE"
    take_fvecs "$QUERY_SRC" "$QUERY_FVECS" "$N_QUERIES"

    log "wrote $BASE_FVECS ($N_BASE vectors)"
    log "wrote $QUERY_FVECS ($N_QUERIES vectors)"
    log "done. Point the benchmark at $BASE_FVECS / $QUERY_FVECS"
}

main "$@"
