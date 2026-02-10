# Third-Party Asset Notices

Last updated: 2026-02-10

This project vendors visual assets under approved licenses and explicit commercial-use terms. No production runtime path may hotlink external asset URLs.

## 1) Heroicons (Icons)

- Name: Heroicons
- Official Source: https://heroicons.com/
- Upstream Repository: https://github.com/tailwindlabs/heroicons
- License: MIT
- License Link: https://github.com/tailwindlabs/heroicons/blob/master/LICENSE
- Version/Tag Used: v2.2.0
- Local Paths:
  - `web/src/assets/icons/heroicons/24/outline/*.svg`

Notes:
- Files were fetched from `src/24/outline` and normalized for design-token use (`currentColor`, unified stroke width).

## 2) unDraw (Illustration Source Files)

- Name: unDraw
- Official Source: https://undraw.co/illustrations
- License Page: https://undraw.co/license
- License Type: Official unDraw license (explicitly permits commercial and non-commercial usage, with stated restrictions)
- Version/Date Acquired: 2026-02-10
- Local Paths:
  - `web/src/assets/illustrations/undraw/raw/process_0wew.svg`
  - `web/src/assets/illustrations/undraw/raw/data-table_xmec.svg`
  - `web/src/assets/illustrations/undraw/raw/secure-usb-drive_7pj5.svg`
  - `web/src/assets/illustrations/undraw/raw/searching-everywhere_tffi.svg`

Restrictions tracked from license page:
- Do not replicate or provide a competing illustration service.
- Do not redistribute assets in packs as a standalone offering.
- Do not perform automated linking/scraping/integration for asset extraction.
- Do not use unDraw assets for AI/ML model training or fine-tuning.

Project policy usage note:
- unDraw files are vendored into the repository and are not hotlinked at runtime.
- Runtime states use project-authored token-aligned SVGs under `web/src/assets/illustrations/states/`.
