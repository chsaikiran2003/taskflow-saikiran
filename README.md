# TaskFlow

A full-stack task management system built for the Zomato — Greening India Tech engineering assignment.

## 1. Overview

TaskFlow lets users register, log in, create projects, and manage tasks with priorities, statuses, and assignees — think a lightweight Jira. The stack is:

- **Backend:** Go 1.22, Chi router, PostgreSQL 16, JWT auth (golang-jwt), bcrypt, golang-migrate, structured logging via `log/slog`
- **Frontend:** React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, React Router v6, react-hot-toast
- **Infrastructure:** Docker Compose, multi-stage Dockerfiles, nginx (frontend), automatic migrations on boot, seed data

---

## 2. Architecture Decisions

### Backend

**Go + Chi** — Chi is a lightweight, idiomatic router that composes well with stdlib `net/http`. It avoids the magic of heavier frameworks while remaining testable. No ORM was used — all SQL is written by hand with `sqlx` for struct scanning. This keeps queries predictable and avoids N+1 surprises.

**Migrations via golang-migrate** — The schema is versioned with explicit up/down migrations. `auto-migrate` was deliberately avoided; schema changes should be reviewable and reversible. Migrations run automatically on container start via the Go entrypoint before the server begins serving.

**JWT in .env** — The JWT secret is loaded from the environment at runtime. It is never hardcoded. Tokens expire after 24 hours and carry `user_id` + `email` claims.

**Passwords** — bcrypt with cost 12 (the minimum required by the spec).

**Structured logging** — `log/slog` (stdlib since Go 1.21) outputs JSON logs with request IDs. No extra dependency needed.

**Graceful shutdown** — The server listens for `SIGTERM` and `SIGINT`, drains in-flight requests with a 30-second timeout, then exits cleanly.

### Frontend

**TanStack Query** — Handles all server state: loading/error/stale states, cache invalidation, and optimistic updates for task status changes. The UI updates immediately on status toggle and reverts on error.

**React Router v6** — Protected routes redirect to `/login` when no token is present. Auth state is persisted in `localStorage` and re-hydrated on refresh.

**Tailwind CSS with custom component classes** — A small set of utility classes (`btn`, `card`, `input`, `badge-*`) keeps the UI consistent without pulling in a full component library. This was a deliberate choice to keep the bundle small and the CSS understandable.

**Dark mode** — Implemented via Tailwind's `class` strategy. The user's preference is persisted in `localStorage` and respects the system preference on first visit.

### What was left out

- **WebSocket / SSE** — real-time updates would require a broker (Redis pubsub or similar). Out of scope for this timeline.
- **Pagination** — the data model supports it but the endpoints don't implement it; added as `?page=&limit=` query params would be the first addition.
- **Refresh tokens** — only access tokens are issued. Adding a refresh token flow and a token blacklist (Redis) would be the production next step.
- **Role-based access** — currently ownership is binary (project owner vs everyone else). A proper RBAC model would be needed for team features.

---

## 3. Running Locally

> Requires: Docker and Docker Compose. Nothing else.

```bash
git clone https://github.com/saikiran/taskflow-saikiran
cd taskflow-saikiran

# Copy environment file
cp .env.example .env

# Start everything (DB + backend + migrations + seed + frontend)
docker compose up --build

# App available at:  http://localhost:3000
# API available at:  http://localhost:8080
```

The first `docker compose up` will:
1. Start PostgreSQL and wait for it to be healthy
2. Build and start the Go backend (which runs migrations automatically)
3. Wait for the backend health check to pass, then run seed data
4. Build and start the React frontend via nginx

Subsequent runs (without `--build`) start in ~5 seconds.

---

## 4. Running Migrations

Migrations run **automatically on backend startup**. No manual step is needed.

If you want to run them manually (e.g. after schema changes during development):

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run up
migrate -path backend/migrations -database "postgres://taskflow:taskflow_secret@localhost:5432/taskflow?sslmode=disable" up

# Run down (rollback all)
migrate -path backend/migrations -database "postgres://taskflow:taskflow_secret@localhost:5432/taskflow?sslmode=disable" down
```

---

## 5. Test Credentials

The seed script creates one user with three tasks across different statuses:

```
Email:    test@example.com
Password: password123
```

---

## 6. API Reference

All protected endpoints require the header:
```
Authorization: Bearer <token>
```

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | Create account |
| POST | `/auth/login` | Sign in, returns JWT |

**POST /auth/register**
```json
// Request
{ "name": "Jane Doe", "email": "jane@example.com", "password": "password123" }

// 201 Response
{ "token": "<jwt>", "user": { "id": "uuid", "name": "Jane Doe", "email": "jane@example.com", "created_at": "..." } }
```

**POST /auth/login**
```json
// Request
{ "email": "jane@example.com", "password": "password123" }

// 200 Response
{ "token": "<jwt>", "user": { ... } }
```

### Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects` | List projects owned by or assigned to current user |
| POST | `/projects` | Create a project |
| GET | `/projects/:id` | Get project + its tasks |
| PATCH | `/projects/:id` | Update name/description (owner only) |
| DELETE | `/projects/:id` | Delete project and all tasks (owner only) |
| GET | `/projects/:id/stats` | Task counts by status and assignee |

**POST /projects**
```json
// Request
{ "name": "Website Redesign", "description": "Optional description" }

// 201 Response
{ "id": "uuid", "name": "Website Redesign", "description": "...", "owner_id": "uuid", "created_at": "..." }
```

**GET /projects/:id**
```json
// 200 Response
{
  "id": "uuid", "name": "...", "owner_id": "uuid",
  "tasks": [
    { "id": "uuid", "title": "...", "status": "todo", "priority": "high", ... }
  ]
}
```

### Tasks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects/:id/tasks` | List tasks (supports `?status=` and `?assignee=` filters) |
| POST | `/projects/:id/tasks` | Create a task |
| PATCH | `/tasks/:id` | Update task fields |
| DELETE | `/tasks/:id` | Delete task (project owner or creator) |

**POST /projects/:id/tasks**
```json
// Request
{ "title": "Design homepage", "description": "...", "priority": "high", "assignee_id": "uuid", "due_date": "2026-04-20" }

// 201 Response — full task object
```

**PATCH /tasks/:id**
```json
// Request — all fields optional
{ "title": "Updated", "status": "done", "priority": "low", "assignee_id": null, "due_date": null }

// 200 Response — updated task object
```

### Error Responses

```json
// 400 Validation error
{ "error": "validation failed", "fields": { "email": "is required" } }

// 401 Unauthenticated
{ "error": "unauthorized" }

// 403 Forbidden
{ "error": "forbidden" }

// 404 Not found
{ "error": "not found" }
```

---

## 7. Running Tests

Integration tests require a running PostgreSQL instance:

```bash
cd backend

# With test DB running (can reuse the docker compose postgres)
TEST_DB_HOST=localhost \
TEST_DB_USER=taskflow \
TEST_DB_PASSWORD=taskflow_secret \
TEST_DB_NAME=taskflow \
go test ./... -v
```

Tests cover: registration, duplicate email, login, wrong password, unauthenticated access, project CRUD, task creation, and validation error shapes.

---

## 8. What I'd Do With More Time

**Shortcuts taken:**
- The `projectMembers` list in the frontend is built from task assignee IDs, not a proper `/projects/:id/members` endpoint. In production, you'd have a membership table.
- Dark mode toggle is present and persisted but the design wasn't fully audited on every screen at every breakpoint.
- No refresh token flow — the JWT access token is the only credential. Expiry is set to 24h which is acceptable for an assignment but not for production.
- The seed password hash was generated once and hardcoded in `seed.sql`. In production you'd generate it at seed time or use a migration tool that supports parameterized seeds.

**What I'd add:**
- Pagination on `/projects` and `/projects/:id/tasks` (`?page=&limit=`)  
- Drag-and-drop task reordering across status columns (react-beautiful-dnd or @dnd-kit)
- WebSocket/SSE for real-time task updates across collaborators
- A proper `/users` endpoint so the assignee dropdown is populated from real project members rather than inferred from existing tasks
- End-to-end tests with Playwright covering the full auth → project → task flow
- Rate limiting on auth endpoints to prevent brute force
- Proper error boundary components in React so errors don't crash the whole UI
