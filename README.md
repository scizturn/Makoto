# Makoto

Birthday email sender for Kyou.id.

Makoto consumes birthday email jobs from Redis, generates a birthday voucher through Kyou.id, sends one of three randomized Kirim.email templates, and posts operational logs to Discord.

## Flow

```text
Redis birthday_email_jobs
  -> Makoto sender
  -> Kyou.id voucher API
  -> Kirim.email validation + send
  -> Discord log
```

`Yukari` is the reader service that reads Hanayo DB and pushes jobs to Redis.

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
MAKOTO_QUEUE_NAME=birthday_email_jobs
MAKOTO_DEAD_LETTER_QUEUE=birthday_email_jobs_dead
MAKOTO_MAX_ATTEMPTS=3

KIRIM_EMAIL_USERNAME=
KIRIM_EMAIL_API_TOKEN=
KIRIM_EMAIL_BASE_URL=https://smtp-app.kirim.email
KIRIM_EMAIL_DOMAIN=kyou.id
KIRIM_EMAIL_VALIDATE=false
KIRIM_EMAIL_FROM_EMAIL=e-support@kyou.id
KIRIM_EMAIL_FROM_NAME=Kyou.id

MAKOTO_TEMPLATE_IDS=birthday1.html,birthday2.html,birthday3.html
MAKOTO_EMAIL_TEMPLATE_DIR=templates/birthday
MAKOTO_EMAIL_SUBJECT='Selamat ulang tahun, {{ .Name }}'
MAKOTO_ACTION_URL=https://kyou.id/account/vouchers
KYOU_ID_API_BASE_URL=https://kyou.id
KYOU_ID_API_TOKEN=

DISCORD_WEBHOOK_ENABLED=true
DISCORD_WEBHOOK_URL=
```

## Redis Job

Makoto expects JSON jobs in `birthday_email_jobs`:

```json
{
  "job_id": "birthday-2026-05-21-user-123",
  "user_id": "123",
  "birthday_date": "2026-05-21T00:00:00+07:00",
  "user": {
    "id": "123",
    "name": "Garvin",
    "email": "garvin@example.com",
    "is_active": true
  },
  "wishlist_items": [],
  "fyp_items": [],
  "popular_items": [],
  "attempt": 1
}
```

Makoto converts wishlist and FYP items into HTML strings before sending. Kirim.email template variables expected:

```text
{{name}}
{{voucher_code}}
{{wishlist_html}}
{{fyp_html}}
{{action_url}}
{{closing}}
```

Wishlist and FYP item payloads can include `image_url`. When present, Makoto renders image cards with public HTTPS image URLs, for example `https://kyoucdn.id/items/example.jpg.webp`.

## Failed Jobs

Makoto retries failed sends until `MAKOTO_MAX_ATTEMPTS`. If the job still fails, Makoto pushes the payload to `MAKOTO_DEAD_LETTER_QUEUE` and logs the failure to Discord.

Retry one dead-letter job manually:

```sh
go run ./cmd/retrydead --job-id birthday-2026-05-21-user-123
```

## Coolify

Deploy this repo as the long-running sender service. Configure Redis and the secrets in Coolify environment variables. Do not commit `.env`.
