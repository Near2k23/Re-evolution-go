<p align="center">
  <a href="https://evolutionfoundation.com.br">
    <img src="./public/hover-evolution.png" alt="Evolution Foundation" />
  </a>
</p>

<h1 align="center">Evolution Go</h1>

<p align="center">
  High-performance WhatsApp API built in Go — part of the Evolution Foundation ecosystem.
</p>

<p align="center">
  <a href="https://github.com/evolution-foundation/evolution-go/releases/latest"><img src="https://img.shields.io/github/v/release/evolution-foundation/evolution-go?include_prereleases&label=version&color=00ffa7" alt="Latest version" /></a>
  <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License: Apache 2.0" /></a>
  <a href="https://docs.evolutionfoundation.com.br"><img src="https://img.shields.io/badge/Docs-evolutionfoundation.com.br-00ffa7" alt="Documentation" /></a>
  <a href="https://evolutionfoundation.com.br/community"><img src="https://img.shields.io/badge/Community-Join%20us-white" alt="Community" /></a>
  <a href="https://hub.docker.com/r/evoapicloud/evolution-go"><img src="https://img.shields.io/badge/Docker-evoapicloud-blue" alt="Docker image" /></a>
</p>

<p align="center">
  <a href="https://evolutionfoundation.com.br">Website</a> &middot;
  <a href="https://docs.evolutionfoundation.com.br">Documentation</a> &middot;
  <a href="https://evolutionfoundation.com.br/community">Community</a> &middot;
  <a href="mailto:suporte@evofoundation.com.br">Support</a>
</p>


---

## About

**Evolution Go** is a high-performance WhatsApp API built in Go. Part of the Evolution Foundation ecosystem, it provides a robust, lightweight solution for WhatsApp integration using the [whatsmeow](https://github.com/tulir/whatsmeow) library.

## Part of the Evolution Foundation ecosystem

Evolution Go is one of the messaging engines maintained by Evolution Foundation. It is used as a WhatsApp provider by the [Evo CRM Community](https://github.com/evolution-foundation/evo-crm-community) and other projects in the ecosystem.

---

## Features

- **High performance** — built with Go for minimal resource usage
- **RESTful API** — clean, well-documented REST endpoints with Swagger
- **Real-time events** — WebSocket, Webhook, AMQP/RabbitMQ and NATS support
- **Media support** — images, videos, audio, documents with MinIO/S3 storage
- **Message storage** — optional PostgreSQL persistence
- **QR code pairing** — built-in QR code generation for device linking
- **Docker ready** — production-ready Docker configuration

---

## Quick Start

### Docker (recommended)

```bash
git clone https://github.com/evolution-foundation/evolution-go.git
cd evolution-go
make docker-build
make docker-run
```

### Local development

```bash
git clone https://github.com/evolution-foundation/evolution-go.git
cd evolution-go

# Setup, configure and run
make setup
cp .env.example .env
make dev
```

> Run `make help` to see all available commands. See [COMMANDS.md](./COMMANDS.md) for detailed workflows.

---

## Configuration

Create a `.env` file:

```env
# Server
SERVER_PORT=8080
CLIENT_NAME=evolution

# Security
GLOBAL_API_KEY=your-secure-api-key-here

# Database
POSTGRES_AUTH_DB=postgresql://postgres:password@localhost:5432/evogo_auth?sslmode=disable
POSTGRES_USERS_DB=postgresql://postgres:password@localhost:5432/evogo_users?sslmode=disable
DATABASE_SAVE_MESSAGES=false

# Logging
DEBUG_ENABLED=DEBUG
LOG_TYPE=console

# Optional
# AMQP_URL=amqp://guest:guest@localhost:5672/
# NATS_URL=nats://localhost:4222
# WEBHOOK_URL=https://your-webhook-url.com/webhook
# MINIO_ENABLED=true
# MINIO_ENDPOINT=localhost:9000
# MINIO_ACCESS_KEY=minioadmin
# MINIO_SECRET_KEY=minioadmin
```

| Variable | Description | Default |
|---|---|---|
| `SERVER_PORT` | Server port | `8080` |
| `CLIENT_NAME` | Client identifier | `evolution` |
| `GLOBAL_API_KEY` | API authentication key | **Required** |
| `DATABASE_SAVE_MESSAGES` | Enable message storage | `false` |
| `DEBUG_ENABLED` | WhatsApp debug level (legacy alias: `WADEBUG`) | `INFO` |

---

## Sin activación de licencia (fork)

Este fork no incluye el gate de licencia de upstream: la API funciona sin
registrar ni activar nada y no se comunica con servidores de licencias.

## Manager

El manager («Enlace») está en `manager/` (Vite + React + TypeScript) y se sirve
en `/manager`. Separa las líneas en conectadas, reconectadas (volvieron solas
tras un corte), reconectando y caídas, a partir de `GET /instance/overview`.

```bash
make manager-install   # una vez
make manager-build     # genera manager/dist, que sirve el binario
make manager-dev       # desarrollo con Vite en :5173 (indica el servidor en el login)
```

La imagen Docker compila el manager en su propio stage; `manager/dist` no se
versiona.

## Webhooks

Cada línea tiene su URL y sus eventos (los fija `/instance/connect`) y, además,
una configuración propia en `GET/PUT /webhook/:instanceId`:

- **Cabeceras adicionales** (por ejemplo `Authorization`).
- **Una URL por evento** (`byEvents`): el evento se añade a la ruta en
  kebab-case (`…/message`, `…/send-message`, `…/connected`).
- **Firma:** con un secreto (`"secret": "new"` lo genera y lo devuelve una sola
  vez), cada envío lleva `X-Webhook-Timestamp` y
  `X-Webhook-Signature: sha256=<hex>`, donde el hex es
  `HMAC_SHA256(secreto, timestamp + "." + cuerpo)`.
- Siempre llegan `X-Webhook-Event` y `X-Webhook-Delivery` (ID único, útil para
  descartar duplicados).

Los envíos fallidos se reintentan con espera creciente (1 s, 2 s, 4 s… hasta
60 s; los 4xx no se reintentan salvo 408, 425 y 429). Las entregas se
registran (`GET /webhook/:instanceId/deliveries`) y `POST
/webhook/:instanceId/test` envía un evento `Test`.

## Historial de chats

Con `CHAT_STORE_ENABLED=true` (por defecto) cada línea guarda sus chats,
contactos y mensajes en Postgres: lo que recibe, lo que envía por la API y el
historial que WhatsApp manda al vincularla. Se consulta en el manager (Chats)
o por la API, con la clave global o el token de la línea:

| Método | Ruta | Uso |
|---|---|---|
| GET | `/store/:instanceId/chats?q=&archived=&before=&limit=` | Chats, fijados primero |
| GET | `/store/:instanceId/messages?chat=<jid>&before=&limit=` | Mensajes, del más nuevo al más antiguo |
| GET | `/store/:instanceId/messages/:messageId` | Un mensaje con su JSON original |
| GET | `/store/:instanceId/contacts?q=` | Contactos conocidos |
| POST | `/store/:instanceId/read` `{"chat":"<jid>"}` | Marca leído y envía los acuses |

Los mensajes se borran pasados `CHAT_STORE_RETENTION_DAYS` días (90 por
defecto; 0 = nunca).

## Bots

Cada línea puede tener varios bots (`/bots/:instanceId`, con la clave global
o el token de la línea; también desde el manager, en **Bots**). En cada chat
habla un solo bot a la vez: el primero, por prioridad, cuyo disparador
coincide (cualquier mensaje o una palabra clave con `contains`, `equals`,
`startsWith`, `endsWith` o `regex`). A partir de ahí la sesión sigue con ese
bot hasta que termina, expira (`expireMinutes`), se escribe la palabra de
cierre (`keywordFinish`) o se pausa.

| Tipo | `config` | Qué hace |
|---|---|---|
| `http` | `url`, `apiKey` | POST JSON con `instanceId`, `remoteJid`, `pushName`, `message`, `messageType`, `messageId`, `fromMe`, `isGroup`, `sessionId`, `botId`. Responde con `message`, `output`, `reply`, `text` o `messages[]` (texto o `{mediaUrl, mediaType}`); `"end": true` cierra la sesión. |
| `n8n` | `url`, `username`, `password` | Igual, contra un nodo Webhook de n8n («Respond to Webhook» con `output`). |
| `openai` | `apiKey`, `model`, `systemPrompt`, `historyLimit`, `baseUrl`, `transcribe` | Chat Completions con el historial reciente; transcribe notas de voz con Whisper. `baseUrl` permite APIs compatibles. |
| `typebot` | `url`, `publicId` | startChat/continueChat; los botones llegan como opciones numeradas. |
| `dify` | `apiUrl`, `apiKey`, `botType` | `chat`, `agent`, `workflow` o `completion`. |
| `flowise` | `url`, `apiKey` | URL de predicción de un chatflow. |

Opciones comunes: `allowGroups` (apagado por defecto), `stopOnFromMe` (pausa
el bot si alguien responde desde el teléfono), `listenFromMe`, `delayMs`
(«escribiendo…»), `debounceMs` (junta mensajes seguidos), `splitMessages`,
`unknownMessage` e `ignoreJids`. Las claves se devuelven enmascaradas.

Sesiones: `GET /bots/:instanceId/sessions` y
`POST /bots/:instanceId/sessions/:sessionId {"status":"paused|open|closed"}`.

## Chatwoot

`GET/PUT /chatwoot/:instanceId` con `url`, `accountId` y `token` (de un
usuario o agent bot de Chatwoot). Al guardar se crea un buzón de tipo API
cuyo webhook apunta a `/chatwoot/webhook/:instanceId` de este servidor. Cada
chat de WhatsApp se vuelve una conversación (con adjuntos) y lo que responde
el agente sale por la línea. Opciones: `signMessages`, `reopenConversation`,
`conversationPending` e `ignoreGroups`. `POST /chatwoot/:instanceId/test`
comprueba la conexión. Detrás de un proxy, el servidor usa
`X-Forwarded-Proto`/`X-Forwarded-Host` para armar la URL del webhook.

---

## API Documentation

Swagger UI available at:

```
http://localhost:8080/swagger/index.html
```

### Key Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/instance/create` | Create WhatsApp instance |
| `GET` | `/instance/{name}/qrcode` | Get QR code for pairing |
| `POST` | `/message/sendText` | Send text message |
| `POST` | `/message/sendMedia` | Send media message |
| `GET` | `/instance/{name}/status` | Get instance status |
| `DELETE` | `/instance/{name}` | Delete instance |

---

## Project Structure

```
evolution-go/
├── cmd/evolution-go/     # Application entry point
├── pkg/
│   ├── instance/        # Instance management
│   ├── message/         # Message handling
│   ├── sendMessage/     # Message sending
│   ├── routes/          # HTTP routes
│   ├── middleware/      # Auth & validation middleware
│   ├── config/          # Configuration
│   ├── events/          # Event producers (AMQP, NATS, Webhook, WS)
│   └── storage/         # Media storage (MinIO)
├── docs/                # Swagger documentation
├── Dockerfile
├── Makefile
└── VERSION
```

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.24+ |
| HTTP framework | Gin |
| WhatsApp | [whatsmeow](https://github.com/tulir/whatsmeow) |
| Database | PostgreSQL |
| ORM | GORM |
| Message queue | RabbitMQ, NATS |
| Object storage | MinIO/S3 |
| Documentation | Swagger/OpenAPI |
| Container | Docker |

---

## Documentation

| Resource | Link |
|---|---|
| Website | [evolutionfoundation.com.br](https://evolutionfoundation.com.br) |
| Documentation | [docs.evolutionfoundation.com.br](https://docs.evolutionfoundation.com.br) |
| Community | [evolutionfoundation.com.br/community](https://evolutionfoundation.com.br/community) |
| Docker Hub | [evoapicloud/evolution-go](https://hub.docker.com/r/evoapicloud/evolution-go) |
| Changelog | [CHANGELOG.md](./CHANGELOG.md) |
| Contributing | [CONTRIBUTING.md](./CONTRIBUTING.md) |
| Security | [SECURITY.md](./SECURITY.md) |

---

## Hosting

Deploy Evolution Go with optimized infrastructure through our HostGator partnership:

[**Evolution Go VPS — HostGator**](https://evolution-api.com/vps-evolution-go)

---

## Telemetry

Evolution Go collects anonymous telemetry data (routes used, API version) to help improve the service. **No sensitive or personal data is collected.**

---

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines on how to submit issues, propose features, and open pull requests.

Join our [community](https://evolutionfoundation.com.br/community) to discuss ideas and collaborate.

---

## Security

For security issues, **do not open a public issue**. Email **suporte@evofoundation.com.br** or use GitHub's private vulnerability reporting. See [SECURITY.md](./SECURITY.md) for details.

---

## Acknowledgments

- [whatsmeow](https://github.com/tulir/whatsmeow) by [Tulir Asokan](https://github.com/tulir) — WhatsApp protocol library
- [Evolution API](https://github.com/evolution-foundation/evolution-api) — Node.js sister project

---

## License

Evolution Go is licensed under the Apache License 2.0, with additional brand-protection conditions (LOGO/copyright preservation and Usage Notification requirement). See [LICENSE](./LICENSE) for full details.

For licensing inquiries, contact **suporte@evofoundation.com.br**.

## Trademarks

"Evolution Foundation", "Evolution" and "Evolution Go" are trademarks of Evolution Foundation. See [TRADEMARKS.md](./TRADEMARKS.md) for the brand assets policy.

Third-party attributions are documented in [NOTICE](./NOTICE).

---

<p align="center">
  Made by <a href="https://evolutionfoundation.com.br">Evolution Foundation</a> · © 2026
</p>
