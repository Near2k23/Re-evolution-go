<h1 align="center">Enlace</h1>

<p align="center">
  <b>A self-hosted WhatsApp API with a modern manager, chat history, bots and Evolution API v2 compatibility.</b><br/>
  Open source · Written in Go · Based on Evolution Go
</p>

<p align="center">
  <b>English</b> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.it.md">Italiano</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Enlace dashboard: operational lines, drops, webhook delivery and hourly availability" width="100%" />
</p>

---

## Why Enlace

Running many WhatsApp numbers is mostly about knowing **which lines are healthy, which ones dropped and came back, and which ones need a person**. Enlace is built around that:

- **See every line at a glance.** Lines are split into *connected*, *reconnected* (dropped and came back on their own), *reconnecting* and *down*, with an hour-by-hour availability map and a log of every cut.
- **Fix them from one place.** Reconnect, link by QR or pairing code, log out, set a proxy or change settings without touching the API.
- **Read and answer chats.** Every line keeps its chats, contacts and messages in PostgreSQL, and the manager shows them like a messaging app.
- **Automate replies.** Bots per line with n8n, OpenAI, Typebot, Dify, Flowise or your own HTTP endpoint, plus two-way Chatwoot.
- **Plug in what you already use.** Signed webhooks with retries, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher and Prometheus metrics.
- **Keep your Evolution API v2 integrations.** The v2 routes and event names work on top of the same server, so n8n, Typebot and existing scripts run unchanged.
- **In your language.** The manager speaks English, Spanish, Portuguese, French, Italian, German, Russian, Chinese, Japanese and Korean, on desktop and mobile.

## Screenshots

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Lines grouped by state" /> | <img src="./docs/images/line-detail.png" alt="Line detail with drops and 7-day connection map" /> |
| **Lines** grouped by state, filterable by your own groups | **Line detail**: drops, longest cut and the last 7 days hour by hour |
| <img src="./docs/images/chats.png" alt="Chats with history" /> | <img src="./docs/images/chat-menu.png" alt="Chat actions" /> |
| **Chats** stored per line, with delivery and read ticks | **Chat actions**: archive, pin, mute, mark as unread, disappearing messages |
| <img src="./docs/images/bots.png" alt="Bots per line" /> | <img src="./docs/images/webhook.png" alt="Webhook settings" /> |
| **Bots** per line, with sessions and Chatwoot | **Webhooks**: events, v2 format, files in base64, signature and headers |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Dashboard on mobile" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Chat on mobile" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Lines in Japanese on mobile" width="30%" />
</p>
<p align="center"><sub>The same manager on a phone, here in English and Japanese.</sub></p>

### Design principles

- **Calm by default, loud when it matters.** Healthy lines fade into the background; drops and lines that need a person come first, with one clear action.
- **No decoration.** No borders or dividers, no gradients: soft surfaces, generous spacing and color only for state (green connected, blue reconnected, orange reconnecting, red down).
- **Readable numbers.** Phone numbers, tokens and times use a monospaced face; counts are large and plain.
- **Same app on every screen.** A sidebar on desktop and a bottom tab bar on mobile, with the same pages and actions.

---

## Quick start

### Docker Compose

```bash
git clone https://github.com/Near2k23/Re-evolution-go.git
cd Re-evolution-go
docker build -t enlace:latest .
```

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: change-me
    volumes: [pgdata:/var/lib/postgresql/data]
  enlace:
    image: enlace:latest
    depends_on: [postgres]
    ports: ["8080:8080"]
    environment:
      SERVER_PORT: "8080"
      GLOBAL_API_KEY: change-me-too
      POSTGRES_AUTH_DB: postgresql://postgres:change-me@postgres:5432/evogo_auth?sslmode=disable
      POSTGRES_USERS_DB: postgresql://postgres:change-me@postgres:5432/evogo_users?sslmode=disable
      DATABASE_SAVE_MESSAGES: "false"
volumes:
  pgdata:
```

Create both databases once (`evogo_auth` and `evogo_users`), start it with `docker compose up -d` and open **http://localhost:8080/manager**. Sign in with the server URL and your `GLOBAL_API_KEY`, create a line and link it with the QR code.

### Local development

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` runs the manager with Vite on port 5173. Run `make help` to see every command.

---

## Features

### Lines and connection health

- `GET /instance/overview` returns every line with its state, drops in the last 24 h, the longest cut and an hourly map. `GET /instance/history/:id` returns the cut log.
- `POST /instance/reconnect/:id` tries to bring a dropped line back without unlinking it.
- Lines start on boot (`CONNECT_ON_STARTUP`, on by default) and can be organized in your own **groups** (`/line-groups`).

### Webhooks

Each line has its URL and events (set with `/instance/connect`) plus its own settings in `GET/PUT /webhook/:instanceId`:

- **Extra headers**, **one URL per event** (`…/message`, `…/send-message`) and **files in base64** (`"base64": true`).
- **HMAC signature**: `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` with `X-Webhook-Timestamp`. Every request also carries `X-Webhook-Event` and a unique `X-Webhook-Delivery`.
- **Retries** with growing waits (1 s, 2 s, 4 s… up to 60 s, 8 attempts; 4xx is not retried except 408/425/429), a delivery log (`/webhook/:id/deliveries`) and a test event (`POST /webhook/:id/test`).
- **Evolution API v2 format** per line or for the global webhook (`WEBHOOK_FORMAT=evolution-api`).

### Chat history

With `CHAT_STORE_ENABLED=true` (default) every line stores its chats, contacts and messages: what it receives, what it sends through the API and the history WhatsApp sends when the line is linked.

| Method | Route | Use |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Chats, pinned first |
| GET | `/store/:id/messages?chat=<jid>&before=` | Messages, newest first |
| GET | `/store/:id/contacts?q=` | Known contacts |
| POST | `/store/:id/read` | Mark as read and send read receipts |

Messages are kept for `CHAT_STORE_RETENTION_DAYS` days (90 by default; 0 = forever).

### Chat actions, groups and contacts

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` with `{"chat": "<jid>"}` or `{"number": "…"}`. `mute` takes `duration` in seconds (8 h by default, `-1` = always).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (disappearing messages), `POST /user/business` (business profile).
- `POST /send/contact` accepts `vcards: [...]` to send several contacts in one message.

### Bots

Several bots per line (`/bots/:instanceId`). In each chat one bot talks at a time: the first one, by priority, whose trigger matches (any message, or a keyword with `contains`, `equals`, `startsWith`, `endsWith` or `regex`).

| Type | What it does |
|---|---|
| `http` | Posts each message as JSON to your URL and replies with what it returns |
| `n8n` | Same, against an n8n Webhook node |
| `openai` | Chat Completions with recent history; transcribes voice notes |
| `typebot` | Runs a published Typebot flow |
| `dify` | Chat, agent, workflow or text-generator apps |
| `flowise` | A Flowise chatflow |

Common options: reply in groups, pause when a person answers, typing delay, grouping of quick messages, splitting long replies, end keyword, expiry and ignored chats.

### Chatwoot

`PUT /chatwoot/:instanceId` with your Chatwoot URL, account and token. Enlace creates an API inbox, mirrors every WhatsApp chat (with attachments) as a conversation and sends agents' replies through the line.

### Evolution API v2 compatibility

The v2 routes live next to the Go ones. Lines are addressed by **name** and `apikey` accepts the global key or the line token.

| Group | Routes |
|---|---|
| Instance | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Messages | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Chat | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Groups | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Settings | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` and `find`) |
| Bots | `typebot`, `openai` (with `creds`), `dify`, `flowise`, `n8n`, `evolutionBot`, and `chatwoot` |

Events can be sent with v2 names and envelope (`messages.upsert`, `connection.update`, …). Differences from v2 are listed in [docs/COMPAT.md](./docs/COMPAT.md).

### Event destinations

Besides webhooks, RabbitMQ, NATS and WebSocket, the events of every line can go to:

| Destination | Main variables |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` or `PUSHER_HOST` |

Each one accepts `<DESTINATION>_EVENTS` (filter) and `<DESTINATION>_FORMAT=evolution-api`.

### Metrics

`GET /metrics` exposes Prometheus metrics: lines by state, messages received and sent, calls, webhook deliveries and delivery time, plus process data. It requires the global key (`apikey` or `Authorization: Bearer`) or basic auth with `METRICS_USER` / `METRICS_PASSWORD`.

---

## Configuration

| Variable | Description | Default |
|---|---|---|
| `GLOBAL_API_KEY` | Admin key for the API and the manager | **required** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | PostgreSQL connection URLs | **required** |
| `SERVER_PORT` | HTTP port | `8080` |
| `CLIENT_NAME` | Groups the lines of this server (useful when several servers share a database) | — |
| `CONNECT_ON_STARTUP` | Reconnect every line on boot | `true` |
| `DATABASE_SAVE_MESSAGES` | Store message delivery status (`true` or `false`) | **required** |
| `CHAT_STORE_ENABLED` | Store chats, contacts and messages | `true` |
| `WEBHOOK_URL` | Global webhook for every line | — |
| `WEBHOOK_FORMAT` | `evolution-api` for the v2 event format | — |
| `SERVER_URL` | Public URL, used in v2 events | — |
| `METRICS_ENABLED` | Turn `/metrics` on or off | `true` |

The full list, with RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher and webhook retries, is in [.env.example](./.env.example).

## API documentation

Swagger UI is served at `/swagger/index.html`. The manager uses the same API, so everything you see in it can be automated.

## Project layout

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Privacy

Enlace does not send telemetry or usage data anywhere. It only talks to WhatsApp and to the destinations you configure.

## Staying in sync with Evolution Go

New features live in their own packages and touch upstream files only through short hooks. [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) explains how to merge new Evolution Go releases.

## Contributing

Issues and pull requests are welcome. Please run `go test ./...` and `make manager-build` before opening a pull request, and keep the manager translations complete in the ten languages.

## Credits and license

Enlace is based on [Evolution Go](https://github.com/evolution-foundation/evolution-go) by Evolution Foundation and uses [whatsmeow](https://github.com/tulir/whatsmeow) by Tulir Asokan.

Licensed under Apache 2.0 with the additional conditions in [LICENSE](./LICENSE). "Evolution Go" is a trademark of Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)); this project is independent and is not affiliated with or endorsed by Evolution Foundation. Third-party notices are in [NOTICE](./NOTICE).
