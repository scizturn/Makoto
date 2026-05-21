# Makoto Sender Implementation Plan

**Goal:** Keep Makoto focused on consuming Redis birthday jobs and sending birthday campaign email.

**Architecture:** Yukari is the DB reader and Redis producer. Makoto is the Redis consumer, Kyou.id voucher client, Kirim.email sender, and Discord logger.

**No commits or pushes:** All files remain uncommitted until explicitly requested.

## Implemented Scope

- Redis job codec and Redis list consumer.
- Full job payload processing without Hanayo DB reads.
- Kyou.id voucher generation before email send.
- Random selection across three Kirim.email template IDs.
- Discord webhook logging for sent/failed jobs.
- Coolify-ready Dockerfile and environment examples.
