# PingMe

PingMe is a Go uptime monitoring backend. Users can register, create URL targets, run periodic HTTP checks, review check logs, and receive downtime or recovery notifications through Telegram or webhooks.

## Features

- JWT authentication with rotating refresh sessions.
- Target management for HTTP and HTTPS URLs.
- Background worker that schedules checks, records results, and tracks monitor state.
- Incident notifications when a target goes down after repeated failures and when it recovers.
- Alert channels for Telegram chats and webhook endpoints.
- Telegram bot deep links for connecting a user's Telegram chat.
- OpenAPI documentation served by the API.

## Tech Stack

- Go 1.22
- Gin HTTP router
- PostgreSQL
- sqlx with the `lib/pq` PostgreSQL driver
- Docker Compose for local services

## Project Structure

```text
cmd/api/             API server and OpenAPI spec
cmd/worker/          Background monitor checker
cmd/bot/             Telegram bot process
internal/            Application packages
migrations/          PostgreSQL migrations
docker-compose.yml   Local PostgreSQL, API, worker, and bot services
Dockerfile           Builds all Go binaries
```

## Requirements

- Go 1.22 or newer
- Docker and Docker Compose
- PostgreSQL client tools are optional if you use the Docker migration command below

## Configuration

Create a local environment file:

```sh
cp .env.example .env
```

Environment variables:

| Variable | Description | Default in `.env.example` |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@db:5432/pingme?sslmode=disable` |
| `JWT_ACCESS_SECRET` | Secret for signing access tokens | `dev_access_secret_change_me` |
| `JWT_REFRESH_SECRET` | Secret for signing refresh tokens | `dev_refresh_secret_change_me` |
| `JWT_ACCESS_TTL` | Access token lifetime | `15m` |
| `JWT_REFRESH_TTL` | Refresh token lifetime | `720h` |
| `HTTP_ADDR` | API listen address | `:8080` |
| `WORKER_COUNT` | Number of checker workers | `4` |
| `WORKER_BATCH_SIZE` | Number of due targets claimed per scheduler tick | `4` |
| `WORKER_QUEUE_SIZE` | Internal worker queue size | `8` |
| `SCHEDULER_TICK` | Scheduler interval | `1s` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token for notifications and bot process | empty |
| `TELEGRAM_BOT_USERNAME` | Telegram bot username used to create deep links | empty |

Use strong JWT secrets outside local development.

## Run with Docker Compose

Start PostgreSQL first:

```sh
docker compose up -d db
```

Apply database migrations:

```sh
for file in migrations/*.up.sql; do
  docker compose exec -T db psql -U postgres -d pingme < "$file"
done
```

Start the API and worker:

```sh
docker compose up --build app worker
```

The API will be available at:

- `http://localhost:8080/health`
- `http://localhost:8080/ready`
- `http://localhost:8080/swagger`
- `http://localhost:8080/openapi.yaml`

If you configured `TELEGRAM_BOT_TOKEN`, start the bot as well:

```sh
docker compose up --build bot
```

To run all configured services at once:

```sh
docker compose up --build
```

## Run Locally

Start the database:

```sh
docker compose up -d db
```

Apply migrations:

```sh
for file in migrations/*.up.sql; do
  docker compose exec -T db psql -U postgres -d pingme < "$file"
done
```

When running Go binaries directly on the host, use `localhost:5433` instead of the Docker service name `db`:

```sh
export DATABASE_URL='postgres://postgres:postgres@localhost:5433/pingme?sslmode=disable'
```

Run the API:

```sh
go run ./cmd/api
```

Run the worker in another terminal:

```sh
go run ./cmd/worker
```

Run the Telegram bot, if needed:

```sh
TELEGRAM_BOT_TOKEN='<your-bot-token>' go run ./cmd/bot
```

## API Overview

Public endpoints:

- `GET /health`
- `GET /ready`
- `GET /swagger`
- `GET /openapi.yaml`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

Authenticated endpoints require an access token:

```http
Authorization: Bearer <access_token>
```

Authenticated endpoints:

- `GET /me`
- `GET /targets`
- `POST /targets`
- `PATCH /targets/{id}`
- `DELETE /targets/{id}`
- `GET /targets/{id}/logs`
- `GET /alert-channels`
- `POST /alert-channels`
- `PATCH /alert-channels/{id}`
- `DELETE /alert-channels/{id}`
- `POST /telegram/link-token`

See the full OpenAPI document at `http://localhost:8080/swagger`.

## Example Flow

Register a user:

```sh
curl -s http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"password123"}'
```

Create a target:

```sh
curl -s http://localhost:8080/targets \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"url":"https://example.com","name":"Example","interval":60,"enabled":true}'
```

Create a webhook alert channel:

```sh
curl -s http://localhost:8080/alert-channels \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"type":"webhook","address":"https://example.com/webhook","enabled":true}'
```

Create a Telegram link:

```sh
curl -s -X POST http://localhost:8080/telegram/link-token \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Open the returned `link_url` in Telegram to connect the chat. Telegram linking requires `TELEGRAM_BOT_USERNAME` in the API process and `TELEGRAM_BOT_TOKEN` in the bot process.

## Worker Behavior

The worker claims due enabled targets, sends HTTP `GET` requests, stores check logs, and updates runtime state.

- `2xx` responses are treated as successful.
- Failed checks increase the target's consecutive failure count.
- After 3 consecutive failures, the target becomes `down` and a `down` event is emitted.
- A successful check after a `down` state marks the target as `up` and emits a `recovered` event.
- Telegram and webhook alert channels receive event notifications when they are enabled.

## Development

Run tests:

```sh
go test ./...
```

Build binaries:

```sh
go build -o api ./cmd/api
go build -o worker ./cmd/worker
go build -o bot ./cmd/bot
```

Stop local services:

```sh
docker compose down
```

Remove local database data:

```sh
docker compose down -v
```
