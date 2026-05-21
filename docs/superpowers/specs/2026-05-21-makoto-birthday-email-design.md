# Makoto Birthday Email Sender Design

## Goal

Makoto is the sender service for Kyou.id birthday sales email automation.

## Architecture

Yukari reads Hanayo DB and pushes complete birthday email jobs to Redis. Makoto consumes those Redis jobs, validates email through Kirim.email, asks Kyou.id to generate a two-week birthday voucher, randomly selects one of three Kirim.email templates, sends the email, and posts operational logs to Discord.

Redis is temporary queue state. Discord is the operational log target. Makoto does not read Hanayo DB.

## Runtime Flow

1. Pop job JSON from Redis key `birthday_email_jobs`.
2. Validate user email with Kirim.email.
3. Send voucher JSON to Kyou.id internal API.
4. Build Kirim.email merge data from the Redis job payload.
5. Randomly select one template from `MAKOTO_TEMPLATE_IDS`.
6. Send the template email through Kirim.email.
7. Send success/failure log to Discord.

## Environment Variables

- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `MAKOTO_QUEUE_NAME`
- `KIRIM_EMAIL_USERNAME`
- `KIRIM_EMAIL_API_TOKEN`
- `KIRIM_EMAIL_BASE_URL`
- `KIRIM_EMAIL_DOMAIN`
- `KIRIM_EMAIL_FROM_EMAIL`
- `KIRIM_EMAIL_FROM_NAME`
- `MAKOTO_TEMPLATE_IDS`
- `KYOU_ID_API_BASE_URL`
- `KYOU_ID_API_TOKEN`
- `DISCORD_WEBHOOK_ENABLED`
- `DISCORD_WEBHOOK_URL`
