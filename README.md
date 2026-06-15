# Makoto

Birthday dan anniversary email sender untuk Kyou.id.

Makoto consume job dari Redis, render template HTML, dan kirim via Kirim.email. Dua worker berjalan concurrent: satu untuk birthday, satu untuk anniversary.

## Flow

```text
Redis (birthday_email_jobs / anniversary_email_jobs)
  -> Makoto worker
  -> Email validation (Kirim.email)
  -> Template render + subject selection
  -> Kirim.email send
  -> Audit log update
  -> Discord log
```

`Yukari` adalah reader service yang membaca Hanayo DB dan mendorong job ke Redis.

## Pipeline

| Pipeline | Queue | Dead Letter | Template Dir |
|---|---|---|---|
| Birthday | `birthday_email_jobs` | `birthday_email_jobs_dead` | `templates/birthday` |
| Anniversary | `anniversary_email_jobs` | `anniversary_email_jobs_dead` | `templates/anniversary` |

Anniversary dikontrol via `MAKOTO_ANNIVERSARY_ENABLED=true`. Kalau false, worker anniversary tidak dijalankan.

## Local Commands

```sh
make test
make build
```

## Environment

```env
MAKOTO_MODE=sender
MAKOTO_TIMEZONE=Asia/Jakarta
MAKOTO_RATE_LIMIT_PER_MINUTE=100

REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

OLD_DATABASE_HOST=mariadb
OLD_DATABASE_PORT=3306
OLD_DATABASE_NAME=kyouid_kyou
OLD_DATABASE_USERNAME=user
OLD_DATABASE_PASSWORD=secret

KIRIM_EMAIL_USERNAME=
KIRIM_EMAIL_API_TOKEN=
KIRIM_EMAIL_BASE_URL=https://smtp-app.kirim.email
KIRIM_EMAIL_DOMAIN=kyou.id
KIRIM_EMAIL_VALIDATE=false
KIRIM_EMAIL_FROM_EMAIL=e-support@kyou.id
KIRIM_EMAIL_FROM_NAME=Kyou.id

MAKOTO_ACTION_URL=https://kyou.id/user/my-voucher
KYOU_ID_API_BASE_URL=https://kyou.id
KYOU_ID_API_TOKEN=

DISCORD_WEBHOOK_ENABLED=true
DISCORD_WEBHOOK_URL=

# Birthday
MAKOTO_QUEUE_NAME=birthday_email_jobs
MAKOTO_DEAD_LETTER_QUEUE=birthday_email_jobs_dead
MAKOTO_MAX_ATTEMPTS=3
MAKOTO_TEMPLATE_IDS=birthday1.html,birthday2.html,birthday3.html
MAKOTO_EMAIL_TEMPLATE_DIR=templates/birthday
MAKOTO_EMAIL_SUBJECT=Tanjoubi Omedetto, {{ .Name }}! Ada hadiah spesial dari Kyou 🎁

# Anniversary
MAKOTO_ANNIVERSARY_ENABLED=true
MAKOTO_ANNIVERSARY_QUEUE_NAME=anniversary_email_jobs
MAKOTO_ANNIVERSARY_DEAD_LETTER_QUEUE=anniversary_email_jobs_dead
MAKOTO_ANNIVERSARY_TEMPLATE_IDS=anniversary1.html,anniversary2.html,anniversary3.html
MAKOTO_ANNIVERSARY_EMAIL_TEMPLATE_DIR=templates/anniversary
# Subject berotasi otomatis (3 variant). Override via MAKOTO_ANNIVERSARY_EMAIL_SUBJECTS
# dengan separator pipe: "Subject 1|Subject 2|Subject 3"
# Variabel tersedia: {{ .Name }}, {{ .FirstName }}, {{ .Years }}
```

## Template Variables

### Birthday

```text
{{ .Name }}          — nama lengkap user
{{ .VoucherCode }}   — kode voucher
{{ .WishlistHTML }}  — grid wishlist (HTML)
{{ .FYPHTML }}       — grid FYP/popular (HTML)
{{ .ActionURL }}     — URL klaim voucher
{{ .Closing }}       — kalimat penutup
```

### Anniversary

```text
{{ .Name }}               — nama lengkap user
{{ .FirstName }}          — nama depan
{{ .VoucherCode }}        — kode voucher
{{ .Years }}              — lama member (tahun)
{{ .JoinDate }}           — tanggal bergabung (misal: "12 Juni 2023")
{{ .AnniversaryEdition }} — misal: "3RD ANNIVERSARY EDITION"
{{ .WishlistHTML }}       — grid wishlist 2x3 (HTML)
{{ .HistoricalHTML }}     — section "X hari yang lalu" (HTML)
{{ .ActionURL }}          — URL klaim voucher
{{ .Closing }}            — kalimat penutup
```

## Subject Rotation

Anniversary menggunakan 3 subject yang dipilih secara deterministik (seed dari tanggal + job ID), sehingga:
- Template dan subject selalu berpasangan (template 1 → subject 1)
- Retry job yang sama menghasilkan subject yang sama
- User berbeda di hari yang sama bisa dapat subject berbeda

Default subjects:
1. `Cieee anniversary! Cek hadiah dari Kyou buat nambahin khilafanmu!`
2. `Ada kado spesial buat anniversary ke-{{ .Years }}, {{ .FirstName }}!`
3. `Kejutan spesial untuk anniversary kamu, {{ .FirstName }}.`

## Redis Job Format

### Birthday

```json
{
  "job_id": "birthday-2026-05-21-user-123",
  "user_id": "123",
  "birthday_date": "2026-05-21T00:00:00+07:00",
  "user": { "id": "123", "name": "Garvin", "email": "garvin@example.com", "is_active": true },
  "voucher_code": "ABCD1234EFGH5678",
  "wishlist_items": [],
  "fyp_items": [],
  "popular_items": [],
  "attempt": 1
}
```

### Anniversary

```json
{
  "job_id": "anniversary-2026-06-12-user-123",
  "user_id": "123",
  "anniversary_date": "2026-06-12T00:00:00+07:00",
  "user": { "id": "123", "name": "Budi Santoso", "email": "budi@example.com", "is_active": true },
  "years": 3,
  "voucher_code": "ANVABCD1234EFGH5",
  "historical_item": {
    "name": "Nendoroid Hatsune Miku",
    "image_url": "https://kyoucdn.id/items/example.webp",
    "order_date": "2023-01-15T00:00:00Z",
    "days_ago": 1244
  },
  "wishlist_items": [],
  "popular_items": [],
  "attempt": 1
}
```

## Failed Jobs

Birthday: retry hingga `MAKOTO_MAX_ATTEMPTS` dengan backoff (5 menit, 15 menit), lalu masuk dead letter queue.

Anniversary: langsung mark failed dan ack — tidak ada retry chain.

Retry manual dari dead letter:

```sh
go run ./cmd/retrydead --job-id birthday-2026-05-21-user-123
```

## Coolify

Deploy sebagai long-running service (container tidak pernah exit). Makoto langsung mulai consume dari Redis saat container start. Pastikan `MAKOTO_ANNIVERSARY_ENABLED=true` di environment Coolify kalau anniversary aktif.
