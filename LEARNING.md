# Makoto Learning Notes

Makoto is the **sender** half of the birthday email system. Yukari reads the database and produces jobs; Makoto consumes those jobs and turns them into real emails.

## System Overview

```text
Hanayo MySQL                                  Redis                                  Internet
┌──────────────┐    SELECT     ┌────────────┐  RPUSH   ┌────────────────┐  BLPOP   ┌────────────┐
│   users      │ ───────────►  │  Yukari    │ ───────► │ birthday_email │ ───────► │  Makoto    │
│   wishlists  │               │ (reader)   │          │   _jobs        │          │  (sender)  │
│   items      │               └────────────┘          └────────────────┘          └─────┬──────┘
│   orders     │                                                                         │
└──────────────┘                                                                         │
                                                                                         ▼
                                                            ┌────────────┐   ┌──────────────────┐
                                                            │  Kyou.id   │   │  Kirim.email     │
                                                            │  voucher   │   │  validate + send │
                                                            └────────────┘   └──────────────────┘
                                                                                         │
                                                                                         ▼
                                                                                 ┌──────────────┐
                                                                                 │  Discord     │
                                                                                 │  webhook log │
                                                                                 └──────────────┘
```

## What Makoto Does

For each birthday job, Makoto:

1. Pops the job from Redis `birthday_email_jobs`.
2. Validates the user's email through Kirim.email strict validation.
3. Asks Kyou.id to generate a personalized voucher code.
4. Picks a random template from `MAKOTO_TEMPLATE_IDS`.
5. Renders wishlist and FYP arrays into HTML strings.
6. Sends the template email through Kirim.email.
7. Posts a success or failure summary to Discord.

If anything fails, the job is logged and skipped; suppressed emails are written back to the suppression flow.

## File Map

```text
cmd/makoto/main.go            CLI entry; wires config, Redis, processor, Discord.
internal/config/config.go     Reads env vars with safe defaults.
internal/domain/types.go      Shared structs: User, WishlistItem, FYPItem, BirthdayJob, EmailMessage.
internal/queue/redis.go       Redis-backed queue (BLPOP / RPUSH).
internal/queue/codec.go       JSON encode/decode for jobs.
internal/queue/memory.go      In-memory queue for tests and old run-once mode.
internal/email/sender.go      Sender + Validator interfaces, local logging fallbacks.
internal/email/kirim.go       Kirim.email REST client (template send + strict validation).
internal/voucher/issuer.go    Voucher Issuer interface + StaticIssuer for local dev.
internal/voucher/kyou.go      Kyou.id voucher HTTP client.
internal/notify/discord.go    Discord webhook logger.
internal/campaign/campaign.go Random template choice + merge data + HTML rendering.
internal/worker/worker.go     Processor that orchestrates the per-job flow.
```

## Concepts Worth Studying

- **Interfaces and seams**: `repository.Store`, `email.Sender`, `email.Validator`, `voucher.Issuer`, `notify.Logger`. Each external system is an interface so tests can swap in fakes.
- **Test-driven development**: every behavior change in this repo started by adding a failing test before the implementation. Read the `*_test.go` files alongside the code.
- **HTML escaping**: `internal/campaign/campaign.go` uses `html.EscapeString` so user-supplied names and URLs cannot break the email template.
- **Random template selection**: `BirthdayCampaign.SelectTemplate` accepts an injectable `RandomIntn` so tests are deterministic while production uses a seeded RNG.
- **HTTP client patterns**: `internal/email/kirim.go` and `internal/voucher/kyou.go` show how to use `http.Client`, `http.NewRequestWithContext`, basic auth, bearer auth, and JSON encode/decode.
- **Redis queue**: `RPUSH` to enqueue and `BLPOP` to wait; `BLPOP` blocks the consumer until a job is available, no busy polling.
- **Rate limiting**: `runSender` in `cmd/makoto/main.go` uses `time.Ticker` so we never exceed the Kirim.email per-minute quota.
- **Email PII masking**: `maskEmail` keeps logs safe in Discord even though they include identifiers.

## Environment

The defaults work for local development without real credentials.

```env
MAKOTO_MODE=sender
MAKOTO_TIMEZONE=Asia/Jakarta
MAKOTO_RATE_LIMIT_PER_MINUTE=100

REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
MAKOTO_QUEUE_NAME=birthday_email_jobs

KIRIM_EMAIL_USERNAME=
KIRIM_EMAIL_API_TOKEN=
KIRIM_EMAIL_BASE_URL=https://smtp-app.kirim.email
KIRIM_EMAIL_DOMAIN=kyou.id
KIRIM_EMAIL_FROM_EMAIL=nandayo@kyou.id
KIRIM_EMAIL_FROM_NAME=Kyou.id

MAKOTO_TEMPLATE_IDS=tpl_001,tpl_002,tpl_003
MAKOTO_ACTION_URL=https://kyou.id/account/vouchers

KYOU_ID_API_BASE_URL=https://kyou.id
KYOU_ID_API_TOKEN=

DISCORD_WEBHOOK_ENABLED=true
DISCORD_WEBHOOK_URL=
```

## Running Locally

1. Start Redis:
   ```sh
   docker run --rm -p 6379:6379 --name makoto-redis redis:7-alpine
   ```
2. In Yukari, push a job (see `Yukari/LEARNING.md`).
3. Run Makoto:
   ```sh
   cd /Users/sleepyreinze/Dev/Makoto
   REDIS_ADDR=127.0.0.1:6379 \
   DISCORD_WEBHOOK_ENABLED=false \
   go run ./cmd/makoto
   ```

Without Kirim.email credentials, Makoto falls back to a logging sender that just prints the rendered template payload. Use that to verify the merge variables before pointing at the real API.

## Tests

```sh
go test ./...
```

Tests in `internal/voucher` and `internal/notify` spin up fake HTTP servers via `httptest`, which is the canonical way to test outbound HTTP clients without leaving the process.

## Kirim.email Template Variables

Bind these in the MJML/HTML template:

```text
{{name}}            user's display name
{{voucher_code}}    code returned from Kyou.id
{{wishlist_html}}   pre-rendered <ul> from wishlist items
{{fyp_html}}        pre-rendered <ul> from FYP fallback items
{{action_url}}      link the CTA button should follow
{{closing}}         personalized closing line
```

`wishlist_html` and `fyp_html` are produced server-side because Kirim.email templates do not loop over arrays.
