# Makoto

Email campaign sender untuk Kyou.id, termasuk discounted wishlist.

Makoto consume job dari Redis, render template HTML, dan kirim via Kirim.email. Worker campaign yang enabled berjalan concurrent.

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
| Discounted wishlist | `discounted_wishlist_email_jobs` | `discounted_wishlist_email_jobs_dead` | `templates/discounted_wishlist` |

Anniversary dikontrol via `MAKOTO_ANNIVERSARY_ENABLED=true`. Kalau false, worker anniversary tidak dijalankan.
Discounted wishlist dikontrol via `MAKOTO_DISCOUNTED_WISHLIST_ENABLED=true` dan default-nya false.

## Local Commands

```sh
make test
make build
go run ./cmd/renderpreview-discounted-wishlist
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
KIRIM_EMAIL_VALIDATION_USERNAME=
KIRIM_EMAIL_VALIDATION_API_TOKEN=
KIRIM_EMAIL_VALIDATION_FAIL_OPEN=false
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

# Discounted wishlist
MAKOTO_DISCOUNTED_WISHLIST_ENABLED=false
MAKOTO_DISCOUNTED_WISHLIST_QUEUE_NAME=discounted_wishlist_email_jobs
MAKOTO_DISCOUNTED_WISHLIST_DEAD_LETTER_QUEUE=discounted_wishlist_email_jobs_dead
MAKOTO_DISCOUNTED_WISHLIST_TEMPLATE_IDS=discounted_wishlist1.html,discounted_wishlist2.html,discounted_wishlist3.html
MAKOTO_DISCOUNTED_WISHLIST_EMAIL_TEMPLATE_DIR=templates/discounted_wishlist
MAKOTO_DISCOUNTED_WISHLIST_URL=https://kyou.id/user/wishlist
# Required when the worker is enabled. Point this at the real preference center.
MAKOTO_DISCOUNTED_WISHLIST_UNSUBSCRIBE_URL=

# Wishlist back-in (queue HARUS sama dgn YUKARI_WISHLIST_BACK_IN_QUEUE_NAME)
MAKOTO_WISHLIST_BACK_IN_ENABLED=false
MAKOTO_WISHLIST_BACK_IN_QUEUE_NAME=wishlist_back_in_email_jobs
MAKOTO_WISHLIST_BACK_IN_DEAD_LETTER_QUEUE=wishlist_back_in_email_jobs_dead
MAKOTO_WISHLIST_BACK_IN_TEMPLATE_IDS=wishlist_back_in1.html,wishlist_back_in2.html,wishlist_back_in3.html
MAKOTO_WISHLIST_BACK_IN_EMAIL_TEMPLATE_DIR=templates/wishlist_back_in
MAKOTO_WISHLIST_BACK_IN_EMAIL_SUBJECT='{{ .FirstName }}, wishlist kamu tersedia lagi!'
# 3 greeting berotasi, separator pipe:
MAKOTO_WISHLIST_BACK_IN_GREETINGS='Omatase, {{ .FirstName }}! Yang kamu tunggu akhirnya balik.|{{ .FirstName }}, penantiannya selesai. Wishlist kamu ready lagi!|Kabar baik, {{ .FirstName }}. Item incaranmu sudah kembali!'
MAKOTO_WISHLIST_BACK_IN_ACTION_URL=https://kyou.id/user/my-voucher
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

### Discounted Wishlist

```text
{{ .FirstName }}            — nama depan user
{{ .DiscountName }}         — nama campaign diskon
{{ .FeaturedHTML }}         — item wishlist utama
{{ .PromoHTML }}            — sisa wishlist + item fill
{{ .WishlistURL }}          — CTA ke wishlist
{{ .FooterHTML }}           — footer campaign
```

### Wishlist Back-In

```text
{{ .FirstName }}         — nama depan user
{{ .Greeting }}          — greeting omatase (berotasi 3 varian)
{{ .BackInItemHTML }}    — list item restock bernomor (01, 02, …) + badge status & harga
{{ .ItemCount }}         — jumlah item di list
{{ .HasVoucher }}        — bool; blok voucher tampil kalau true
{{ .VoucherCode }}       — kode voucher
{{ .ActionURL }}         — URL klaim voucher
{{ .HasCompanion }}      — bool; section "Gas, Nemenin…" tampil kalau true
{{ .CompanionName }}     — nama item yang sudah dibeli (referensi header)
{{ .RecoHTML }}          — grid 6 item cross-sell most-popular (HTML)
{{ .Closing }}           — kalimat penutup
{{ .FooterHTML }}        — footer campaign
```

Badge status (`ready`/`PO`/`LPO`/`BO`/Revive) + harga (diskon dengan coret asli, atau `DP IDR <dp> / <full>` untuk PO, else polos) dirender di dalam `{{ .BackInItemHTML }}` & `{{ .RecoHTML }}` — persis mengikuti hanamaru. Preview tiga template: `go run ./cmd/renderpreview-wishlist-back-in`.

Template, subject, dan greeting dipilih deterministik dari tanggal + job ID, sehingga retry job yang sama tetap konsisten. Makoto hanya mengirim jika `user.is_active=true`, email tidak kosong, dan validator email menerima alamat tersebut.

Preview seluruh tiga template:

```sh
go run ./cmd/renderpreview-discounted-wishlist
```

Output ditulis ke `templates/preview/discounted-wishlist*-preview.html`.

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

### Discounted Wishlist

```json
{
  "job_id": "discounted-wishlist-2026-06-20-user-123",
  "user_id": "123",
  "date": "2026-06-20T00:00:00+07:00",
  "user": { "id": "123", "name": "Budi Santoso", "email": "budi@example.com", "is_active": true },
  "items": [
    {
      "id": "1001",
      "name": "Nendoroid Example",
      "character_name": "Example",
      "url": "https://kyou.id/items/1001/",
      "image_url": "https://kyoucdn.id/example.webp",
      "original_price": 750000,
      "discount_price": 600000,
      "discount_name": "Mid Year Sale",
      "status": "ready",
      "is_wishlisted": true
    }
  ],
  "attempt": 1
}
```

## Failed Jobs

Birthday: retry hingga `MAKOTO_MAX_ATTEMPTS` dengan backoff (5 menit, 15 menit), lalu masuk dead letter queue.

Anniversary: langsung mark failed dan ack — tidak ada retry chain.

Discounted wishlist retry hingga `MAKOTO_MAX_ATTEMPTS` dengan backoff 5 dan 15 menit. Attempt terakhir dipindahkan ke `MAKOTO_DISCOUNTED_WISHLIST_DEAD_LETTER_QUEUE`, baru payload processing di-ack.

## Discounted Wishlist Release Checklist

Code blocker sebelumnya sudah ditangani:

- send/validation/render error menggunakan retry dan dead letter;
- inactive user, email kosong, dan email yang ditolak validator dicatat `skipped`, bukan `sent`;
- processor dan failure handler memiliki test untuk send, skip, retry, dan dead letter;
- worker menolak start ketika enabled tanpa absolute HTTP(S) `MAKOTO_DISCOUNTED_WISHLIST_UNSUBSCRIBE_URL`, dan footer menampilkan link preference center tersebut.

Sebelum mass-send, tim tetap harus:

- mengganti contoh URL dengan preference center Kyou yang benar-benar menyimpan opt-out;
- mengirim seed email dan memeriksa Gmail, Outlook, serta mobile;
- memastikan unsubscribe webhook/suppression Kirim.email dikonfigurasi di environment provider. Transactional API yang dipakai repo ini tidak mendokumentasikan field custom `List-Unsubscribe`, jadi jangan mengasumsikan header tersebut aktif tanpa verifikasi provider.

Retry manual dari dead letter:

```sh
go run ./cmd/retrydead --job-id birthday-2026-05-21-user-123
```

## Coolify

Deploy sebagai long-running service (container tidak pernah exit). Makoto langsung mulai consume dari Redis saat container start. Queue name Yukari dan Makoto harus sama. Enable discounted wishlist hanya setelah release checklist selesai.
