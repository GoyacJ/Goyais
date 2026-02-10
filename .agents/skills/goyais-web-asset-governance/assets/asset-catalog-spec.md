# RESOURCE_CATALOG Spec

Catalog file: `web/src/assets/RESOURCE_CATALOG.yaml`

Each entry must include:

- `asset_id`
- `type` (`icon` | `illustration` | `background`)
- `scenario` (usage context such as `commands-empty`, `nav-home`)
- `local_path` (repo-relative path)
- `license`
- `source_url`
- `version_or_date`
- `token_constraints` (for example `currentColor`, stroke width, class binding)

Rules:

- `local_path` must point to an existing file in repo.
- One catalog entry per concrete asset file.
- Third-party sources must include a license family and retrieval version/date.
