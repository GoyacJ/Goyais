#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MONOREPO_ROOT="$(cd "$ROOT_DIR/../.." && pwd)"
SCHEMA_DIR="$ROOT_DIR/schemas/v2"
OUT_DIR="$ROOT_DIR/generated/python/src/goyais_protocol"
OUT_FILE="$OUT_DIR/models.py"
PROJECT_VERSION="$(node -e 'const fs=require("node:fs"); const path=require("node:path"); const pkg=JSON.parse(fs.readFileSync(path.resolve(process.argv[1], "package.json"), "utf8")); process.stdout.write(pkg.version);' "$MONOREPO_ROOT")"

mkdir -p "$OUT_DIR"

if command -v datamodel-codegen >/dev/null 2>&1; then
  datamodel-codegen \
    --input "$SCHEMA_DIR/event-envelope.schema.json" \
    --input-file-type jsonschema \
    --output "$OUT_FILE" \
    --target-python-version 3.11
else
  cat > "$OUT_FILE" <<'PYEOF'
"""Auto-generated placeholder for protocol models.
Install datamodel-code-generator to regenerate strict models.
"""

from pydantic import BaseModel


class EventEnvelope(BaseModel):
  protocol_version: str
  trace_id: str
  event_id: str
  execution_id: str
  seq: int
  ts: str
  type: str
  payload: dict
PYEOF
fi

cat > "$OUT_DIR/__init__.py" <<'PYEOF'
from .models import *
PYEOF

cat > "$ROOT_DIR/generated/python/pyproject.toml" <<PYEOF
[project]
name = "goyais-protocol"
version = "${PROJECT_VERSION}"
description = "Generated protocol models for Goyais"
requires-python = ">=3.11"
dependencies = ["pydantic>=2.10.0"]
PYEOF

echo "generated $OUT_FILE"
