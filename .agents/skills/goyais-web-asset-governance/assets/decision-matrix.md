# Decision Matrix

## 1) Resource already exists in catalog?

- Yes: reuse and stop.
- No: continue to source selection.

## 2) Source in approved whitelist?

- Yes: fetch and vendor locally.
- No: reject and pick a compliant source.

## 3) Runtime asset token-aligned?

- Yes: keep.
- No: normalize to `currentColor`/token-driven styles.

## 4) Documentation updated?

- `RESOURCE_CATALOG.yaml`: required
- `THIRD_PARTY_NOTICES.md`: required for third-party assets

## 5) Validation passed?

- Run `.agents/skills/goyais-web-asset-governance/scripts/validate-assets.sh`
- If failed, block delivery.
