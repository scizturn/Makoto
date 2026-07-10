# Birthday Email Templates

Makoto can render your own HTML and send the final result through Kirim.email transactional API v4.

Set these env values:

```env
MAKOTO_TEMPLATE_IDS=birthday1.html,birthday2.html,birthday3.html
MAKOTO_EMAIL_TEMPLATE_DIR=templates/birthday
```

Subjects are not configurable via env — they live in `internal/config/config.go`, one per template.

Available variables:

- `{{ .Name }}`: user display name.
- `{{ .VoucherCode }}`: generated birthday voucher code, or `LOCAL-BIRTHDAY` when voucher API is disabled.
- `{{ .WishlistHTML }}`: ready-to-insert wishlist HTML. It renders image cards when `image_url` exists.
- `{{ .FYPHTML }}`: ready-to-insert FYP HTML, with popular fallback handled before rendering. It renders image cards when `image_url` exists.
- `{{ .ActionURL }}`: Kyou voucher/action page.
- `{{ .Closing }}`: configured closing sentence.

Keep the filenames in `MAKOTO_TEMPLATE_IDS`. Makoto randomizes one of those files for every birthday job.
