# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

课程推荐智能体系统 — An AI-powered course recommendation and knowledge graph system for AI curriculum learning. The system combines a Go backend API, Vue 3 frontend, Neo4j knowledge graph, and DeepSeek AI integration.

## Common Commands

### Backend (Go + Gin)

```bash
cd backend
go run ./cmd/server          # Start dev server (default :8080)
go test ./...                # Run all tests
go test ./internal/module/graph/service/...  # Run a single package's tests
go build -o server ./cmd/server              # Build binary
```

### Frontend (Vue 3 + Vite)

```bash
cd frontend
npm install                  # Install dependencies
npm run dev                  # Start dev server (proxies /api to localhost:8080)
npm run build                # Build to ../backend/web/dist (Go serves it)
```

### Docker Full Stack

```bash
docker compose -f docker/neo4j/docker-compose.yml up -d --build
# Brings up: MySQL, Redis, Neo4j (with seed data), Go backend
# Health checks: GET /healthz, GET /readyz
```

## Architecture

### Backend Module System

The backend follows a strict **module pattern** under `backend/internal/module/`. Each module is a self-contained vertical slice:

```
module/<name>/
  dto/        # Request/response types
  entity/     # Database/domain models
  handler/    # Gin HTTP handlers
  repository/ # Data access (MySQL via GORM, Neo4j, Redis)
  service/    # Business logic
  module.go   # Wires repo → service → handler, exposes RegisterRoutes(gin.RouterGroup)
```

**Modules**: auth, graph, graphwriter, recommend, competency, dashboard, frontier, learning, teacher, assistant, user

### Bootstrap & Wiring

`bootstrap.New()` in `backend/internal/bootstrap/app.go` is the composition root:
1. Loads config from env vars (`.env` file, no config files)
2. Connects infrastructure (MySQL/GORM, Redis, Neo4j)
3. Instantiates all modules with their dependencies
4. Builds the Gin router via `router.New()` which mounts middleware and module routes

Route registration is split: public routes (auth login, health) vs auth-guarded routes (everything else via `middleware.Auth`).

### Frontend Architecture

- **Router**: `frontend/src/router.js` — Vue Router with `requiresAuth` / `guestOnly` route guards
- **State**: Pinia stores in `frontend/src/stores/` — `userStore` (auth bootstrap), `graphStore` (graph state)
- **API layer**: `frontend/src/api/request.js` — `request()` for JSON calls, `streamRequest()` for SSE streams; all calls go to `/api/*` which Vite proxies to the Go backend in dev
- **Graph visualization**: AntV G6 (`@antv/g6`) renders the knowledge graph in `components/graph/GraphCanvas.vue`
- **AI chat**: Composable `useAiChat.js` handles streaming SSE from `/api/assistant/chat`
- **Math rendering**: KaTeX via `useKatex.js` composable

### Data Flow

- Frontend build output goes to `backend/web/dist` — the Go server serves it as static files in production
- Vite dev server proxies `/api/*` to Go backend (configurable via `VITE_API_PROXY_TARGET`)
- Backend uses DeepSeek API for AI features (chat, content generation) — configured via `DEEPSEEK_*` env vars
- Knowledge graph data lives in Neo4j; `graphwriter` module asynchronously writes topology updates
- MySQL stores users, classrooms, questions, learning progress; Redis for caching and pub/sub

### Shared Packages

`backend/pkg/` contains cross-cutting utilities:
- `apperr` — structured application errors
- `result` — standardized JSON response wrapper (`{code, message, data}`)
- `ctxmeta` — request context metadata (trace ID, user info)
- `logger` — zap-based structured logger with context support
- `password` — bcrypt password hashing

### Key Config

All configuration is via **environment variables** (loaded from `.env` at project root). See `.env.example` for the full list. Critical vars: `MYSQL_DSN`, `REDIS_ADDR`, `NEO4J_URI`, `JWT_SECRET`, `DEEPSEEK_API_KEY`.

### Knowledge Graph Pipeline

`pipelines/txt2neo4j/` — Python pipeline that extracts structured knowledge points from textbook content and imports them into Neo4j. Separate from the Go backend.

### Database Migrations

SQL migration files live in `backend/migrations/` and are applied automatically by Docker init scripts. They are numbered sequentially (000001, 000002, ...).
