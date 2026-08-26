# RingRouter

Self-deployed LLM API Gateway. Route requests to multiple AI providers through a unified OpenAI-compatible API.

## Tech Stack

- **Backend**: Go 1.27+, net/http, GORM v2
- **Database**: SQLite, MySQL, PostgreSQL
- **Frontend**: (coming soon) React + TypeScript + Vite

## Quick Start

```bash
cp .env.example .env
# Edit .env with your OpenAI API key
make build
./RingRouter
```

Test:
```bash
curl http://localhost:3000/health
curl http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer <your-admin-key>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}'
```

## Architecture

```
Request → Auth → Router → Provider(OpenAI/Claude/...) → Upstream
                ↓
              SQLite/MySQL/PG
```

## License

MIT