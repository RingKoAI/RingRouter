<p align="right">
  <strong>English</strong>
</p>

<div align="center">

# RingRouter

_✦ One gateway, every provider — multi-protocol in, multi-channel out, self-healing LLM API gateway ✦_

</div>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue" alt="license"></a>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go" alt="go">
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react" alt="react">
  <img src="https://img.shields.io/badge/docker-compose-ready-2496ED?logo=docker" alt="docker">
</p>

<p align="center">
  <a href="#features">Features</a> ·
  <a href="#quick-start">Quick Start</a> ·
  <a href="#environment-variables">Environment</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#differences-from-one-api">vs one-api</a> ·
  <a href="#development">Development</a>
</p>

> [!NOTE]
> Self-hosted gateway. Users must follow each upstream provider's terms of service and local laws.

## Features

**Protocol layer**
- [x] Four inbound wire formats, any-to-any (unified pivot conversion): OpenAI Chat Completions / OpenAI Responses / Anthropic Messages / Google Gemini `generateContent`
- [x] Streaming (SSE) and non-streaming; same-protocol streams pass through untouched, cross-protocol streams are translated automatically
- [x] `GET /v1/models` aggregated model listing across channels

**Routing layer**
- [x] Multi-channel: model matching → priority ordering → failover across candidates
- [x] Channel keys sealed with AES-GCM at rest; never echoed back by the admin API
- [x] Model mapping (client model name → upstream model name, JSON config)
- [x] Channel cache: 30s in-process snapshot; optional shared Redis snapshot (cross-instance consistency, invalidated on write)

**Users & billing**
- [x] Groups: `name / uuid / metadata / ratio` (billing multiplier); channels may join multiple groups (comma-separated); renames cascade
- [x] Plans & subscriptions: a plan bundles quota + group + duration; granting applies instantly; subscriptions are snapshot records with lazy expiry
- [x] API keys (`sk-rr-` prefix): revealed exactly once at creation, masked afterwards
- [x] Request logs: model / token counts / latency / channel / IP persisted asynchronously; personal and admin views

**Authentication**
- [x] Username + password login; email-code password reset (60s cooldown, 5 attempts, single use)
- [x] Passkeys (WebAuthn): passwordless sign-in, on-device enrollment, discoverable flow
- [x] Cloudflare Turnstile (optional)
- [x] Admin key (`ADMIN_KEY`) exchanges for an admin session

**Management**
- [x] 4-step setup wizard (site info → SMTP → passkeys → usage mode)
- [x] Full admin pages: channels / users / groups / plans / subscriptions / model catalogue / logs / settings
- [x] Public model plaza at `/models` — browse models and group ratios without signing in
- [x] Built-in streaming playground
- [x] Four UI languages (zh / zh-TW / zh-HK / en), dark mode, single-binary deploy (frontend embedded)

**Not yet implemented** (PRs welcome): weighted load balancing, per-request quota deduction, per-model pricing, redemption codes, referral rewards, OAuth2 sign-in

## Quick Start

### Docker Compose (recommended, PostgreSQL + Redis)

```shell
git clone https://github.com/RingKoAI/RingRouter.git
cd RingRouter
cp .env.example .env          # set ADMIN_KEY / JWT_SECRET / DB passwords
docker compose up -d --build
```

Open `http://localhost:3000` and complete the setup wizard (creates the administrator).

### Docker single container (SQLite lightweight mode)

Build the image from source first:

```shell
docker build -t ringrouter .
docker run --name ringrouter -d --restart always -p 3000:3000 \
  -e ADMIN_KEY=change-me -e JWT_SECRET=change-me-too \
  -e DB_TYPE=sqlite -v /data/ringrouter:/app/data \
  ringrouter
```

### From source

```shell
git clone https://github.com/RingKoAI/RingRouter.git
cd RingRouter

# frontend
cd web && pnpm install && pnpm build && cd ..

# backend (embeds web/dist into a single binary)
go build -o ringrouter .
ADMIN_KEY=change-me ./ringrouter
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Listen port | `3000` |
| `DB_TYPE` | `postgres` / `mysql` / `sqlite` | `postgres` |
| `DB_DSN` | PG / MySQL connection string | — |
| `DB_PATH` | SQLite file path (sqlite only) | `data/ringrouter.db` |
| `ADMIN_KEY` | Bootstrap admin key, exchanges for an admin session | random |
| `JWT_SECRET` | Salt for key sealing & signing (AES-GCM derivation) | random (rotates on restart — pin it) |
| `REDIS_CONN_STRING` | `redis://[user[:pass]@]host:port/db`; presence enables the shared cache | disabled |
| `REDIS_ENABLED` etc. | Alternative discrete form: `REDIS_ENABLED=true` + `REDIS_ADDR/PASSWORD/DB` | disabled |
| `OPENAI_API_KEY` / `OPENAI_BASE_URL` | Fallback upstream when no DB channel matches (optional) | — |
| `ANNOUNCEMENT` | Announcement seeded on first boot (optional) | — |
| `TURNSTILE_SITEKEY` / `TURNSTILE_SECRET` | Cloudflare Turnstile | disabled |

> [!IMPORTANT]
> Pin `JWT_SECRET` in production (the random default rotates per restart, which invalidates sealed channel keys), and serve over HTTPS.

## Usage

1. **Setup wizard**: first visit lands on `/setup` — create the administrator, optionally configure SMTP and passkeys
2. **Add a channel**: dashboard → channels → upstream protocol (openai / anthropic / google), base URL, key, model list, group and priority
3. **Create a key**: dashboard → API keys → generate an `sk-rr-…` (shown once)
4. **Call the gateway** (any protocol, any compatible client):

```bash
# OpenAI protocol
curl http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-rr-xxxx" -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}'

# Anthropic protocol (same key)
curl http://localhost:3000/v1/messages \
  -H "x-api-key: sk-rr-xxxx" -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-5","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}'

# Gemini protocol (same key)
curl "http://localhost:3000/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "x-goog-api-key: sk-rr-xxxx" \
  -d '{"contents":[{"parts":[{"text":"hi"}]}]}'
```

5. **Model plaza**: `/models` is public — browse every available model and group ratio without an account

## Differences from one-api

RingRouter is inspired by and pays respect to [one-api](https://github.com/songquanpeng/one-api) (group ratios, channel priorities, and optional-Redis semantics stay aligned). Key differences:

| Aspect | RingRouter | one-api |
|--------|-----------|---------|
| Inbound protocols | Four protocols, any-to-any (incl. Responses / Gemini) | Mostly OpenAI-compatible |
| Groups | First-class entity (uuid / metadata / ratio) + multi-group channels | String convention + ratio config |
| Subscriptions | Plan/subscription snapshots with lifecycle | None (quota top-up model) |
| Auth | Password + email codes + Passkeys + Turnstile | Password + email + several OAuth |
| Billing | Plan-grant model (quota/ratios ready, per-request deduction TODO) | Full quota system |

## Development

```bash
# backend (Go 1.27)
go build ./... && go vet ./... && go test ./internal/...

# frontend (web/, pnpm)
cd web && pnpm install && pnpm dev   # Vite :5173, proxies to :3000
```

Layout: `internal/` backend (config / crypto / database / gateway / handler / inbound / middleware / model / provider / setting / turnstile / cache), `web/` frontend (React 19 + Vite + Tailwind v4). UI translations live in `web/src/i18n/locales` (zh / zh-TW / zh-HK / en — keep all four in sync).

## License

[AGPL-3.0](./LICENSE)
