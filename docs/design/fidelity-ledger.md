# WeCratfs visual fidelity ledger

The generated concept image at
[`wecratfs-storefront-concept.png`](wecratfs-storefront-concept.png) is a
directional visual reference for the first prototype, not a source of product
records. The implementation keeps the approved brand promise and empty-state
truthfulness while avoiding invented catalogue imagery.

| Reference intent | Implementation | Decision |
| --- | --- | --- |
| Handmade and organic positioning | “Closer to nature, kinder to everyday life” hero copy and values section | Matched |
| Warm paper, indigo, citrus, and terracotta palette | Checked-in embedded stylesheet at `internal/transport/web/static/app.css` | Matched |
| Editorial, spacious storefront composition | Responsive two-column hero, wide content container, restrained cards | Matched |
| Natural-material visual language | Material-first hero panel and no-placeholder product media state | Matched without fake product photography |
| Nature-inspired botanical artwork | CSS-native hero panel instead of a product or brand image | Intentional deviation until approved brand artwork exists |
| Wordmark shown in generated concept | Runtime wordmark is the exact user-provided name `WeCratfs` | Corrected; generated concept text is not authoritative |
| Catalogue cards in the concept | Cards render only approved PostgreSQL records; empty database shows an explicit state | Corrected for real-data requirement |

## Capture evidence

The responsive preview was captured locally with installed headless Chrome at
1440×1100 and 390×844. The browser connector was unavailable in this
environment, so this is headless browser evidence rather than a live browser
connector session. The capture command is:

```sh
google-chrome --headless=new --no-sandbox --disable-gpu \
  --window-size=1440,1100 \
  --screenshot=/tmp/wecratfs-desktop.png \
  http://localhost:8080/
```

The same command with `--window-size=390,844` produced the mobile capture.
