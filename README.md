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

Con `"base64": true` (o «Incluir archivos» en el manager) la media recibida y
enviada llega en base64 dentro del evento; `WEBHOOK_FILES=true` lo activa para
todas las líneas.

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

## SQS, Kafka y Pusher

Además del webhook, RabbitMQ, NATS y WebSocket, el servidor puede enviar los
eventos de todas las líneas a:

| Destino | Variables |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL` (una cola; `.fifo` agrupa por línea), `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` (o las credenciales por defecto de AWS), `SQS_ENDPOINT` (compatibles) |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC` (`evolution-events`), `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_MECHANISM` (`plain`, `scram-sha-256`, `scram-sha-512`), `KAFKA_SASL_USERNAME`, `KAFKA_SASL_PASSWORD`, `KAFKA_TLS` |
| Pusher | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` o `PUSHER_HOST` (Soketi), `PUSHER_CHANNEL` (por defecto, el nombre de la línea) |

Cada destino acepta `<DESTINO>_EVENTS` (grupos como `MESSAGE,CONNECTION` o
nombres de evento; vacío = todos) y `<DESTINO>_FORMAT=evolution-api`. Kafka usa
el ID de la línea como clave, así que los eventos de una línea llegan en orden.
Pusher admite hasta 10 KB: un evento mayor llega como aviso con
`"truncated": true`. Los envíos fallidos se reintentan 3 veces.

## Métricas

`GET /metrics` devuelve métricas en formato Prometheus. Pide la clave global
(cabecera `apikey` o `Authorization: Bearer`) o, si se definen `METRICS_USER`
y `METRICS_PASSWORD`, autenticación básica. `METRICS_ENABLED=false` lo apaga.

```yaml
scrape_configs:
  - job_name: evolution
    metrics_path: /metrics
    scheme: https
    basic_auth: { username: prometheus, password: "…" }
    static_configs: [{ targets: ["api.ejemplo.com"] }]
```

| Métrica | Qué mide |
|---|---|
| `evolution_instances{state}` | Líneas por estado: `open`, `connecting`, `close` |
| `evolution_instance_up{instance}` | 1 si la línea está conectada |
| `evolution_messages_received_total{instance}` / `evolution_messages_sent_total{instance}` | Mensajes recibidos y enviados por la API desde el arranque |
| `evolution_calls_received_total{instance}` | Llamadas recibidas |
| `evolution_webhook_deliveries_total{result}` | Entregas de webhook `ok` o `failed` |
| `evolution_webhook_delivery_seconds` | Histograma del tiempo de entrega (con reintentos) |
| `evolution_build_info{version}`, `go_*`, `process_start_time_seconds` | Versión y proceso |

## Chats, grupos y contactos

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` con
  `{"chat": "<jid>"}` o `{"number": "57300…"}`. `mute` acepta `duration` en
  segundos (8 h por defecto, `-1` = siempre) y `markunread` acepta
  `"read": true`. Archivar y marcar usan el último mensaje del historial de
  chats, que WhatsApp exige.
- `POST /group/inviteinfo {"code"}`: datos del grupo de un enlace sin unirse.
- `POST /group/ephemeral {"groupJid", "expiration"}`: mensajes temporales
  (0, 86400, 604800 o 7776000 segundos).
- `POST /user/business {"number"}`: perfil de empresa.
- `POST /send/contact` acepta `vcards: [...]` para varios contactos en un
  mensaje, y `email` y `url` en cada uno.

## Compatibilidad con Evolution API v2

Las integraciones hechas para Evolution API v2 (n8n con
`n8n-nodes-evolution-api`, Typebot, scripts propios) funcionan contra este
servidor sin cambios: las rutas v2 conviven con las de Evolution GO, que siguen
igual. La línea se identifica por su **nombre** (`:instanceName`) y la cabecera
`apikey` acepta la clave global o el token de la línea.

| Grupo | Rutas |
|---|---|
| Instancia | `POST /instance/create`, `GET /instance/fetchInstances`, `GET /instance/connect/:n`, `GET /instance/connectionState/:n`, `POST\|PUT /instance/restart/:n`, `DELETE /instance/logout/:n`, `DELETE /instance/delete/:n`, `POST /instance/setPresence/:n` |
| Mensajes | `POST /message/{sendText, sendMedia, sendPtv, sendWhatsAppAudio, sendSticker, sendLocation, sendContact, sendReaction, sendPoll, sendList, sendButtons, sendStatus}/:n` |
| Chat | `POST /chat/{whatsappNumbers, markMessageAsRead, updateMessage, sendPresence, fetchProfilePictureUrl, fetchProfile, updateBlockStatus, findChats, findMessages, findContacts, getBase64FromMediaMessage, updateProfileName, updateProfileStatus, updateProfilePicture, archiveChat, markChatUnread, fetchBusinessProfile}/:n`, `DELETE /chat/deleteMessageForEveryone/:n`, `GET /chat/{findChatByRemoteJid, fetchPrivacySettings}/:n` |
| Grupos | `POST /group/{create, updateGroupSubject, updateGroupDescription, updateGroupPicture, updateSetting, updateParticipant, revokeInviteCode, toggleEphemeral}/:n`, `GET /group/{fetchAllGroups, findGroupInfos, participants, inviteCode, inviteInfo, acceptInviteCode}/:n`, `DELETE /group/leaveGroup/:n` |
| Etiquetas | `GET /label/findLabels/:n`, `POST /label/handleLabel/:n` |
| Ajustes | `POST /settings/set/:n`, `GET /settings/find/:n`, `POST /proxy/set/:n`, `GET /proxy/find/:n`, `POST /webhook/set/:n`, `GET /webhook/find/:n`, `POST /{websocket, rabbitmq, nats}/set/:n`, `GET /{websocket, rabbitmq, nats}/find/:n` |
| Bots | `/{typebot, openai, dify, flowise, n8n, evolutionBot}/{create, find, fetch/:id, update/:id, delete/:id, fetchSessions/:id, changeStatus, fetchSettings}/:n`, `/openai/creds/:n` (y `DELETE /openai/creds/:id/:n`) |
| Chatwoot | `POST /chatwoot/set/:n`, `GET /chatwoot/find/:n` |

Las respuestas imitan las de v2 (`key{remoteJid, fromMe, id}`, `message`,
`messageType`, `messageTimestamp`, `status`…) y los errores usan
`{status, error, response:{message}}`. `findChats`, `findMessages` y
`findContacts` leen el historial de chats (`CHAT_STORE_ENABLED`).

**Eventos.** Un webhook configurado con `/webhook/set` (o en el manager, con
«Formato Evolution API v2») recibe los eventos con los nombres y el sobre de v2:
`{event: "messages.upsert", instance, data, destination, date_time, sender,
server_url, apikey}`. Con «webhook por eventos» la URL termina en
`/messages-upsert`, `/connection-update`… Equivalencias: `Message` →
`messages.upsert`, `SendMessage` → `send.message`, `Receipt` →
`messages.update`, `QRCode` → `qrcode.updated`, `Connected`/`LoggedOut`… →
`connection.update`, `Presence`/`ChatPresence` → `presence.update`, llamadas
→ `call`, contactos → `contacts.update`, grupos → `groups.upsert|update`,
etiquetas → `labels.edit|association`. Los eventos sin equivalente no se
envían en este formato. `WEBHOOK_FORMAT=evolution-api` aplica lo mismo al
webhook global; el resto de líneas sigue con el formato de Evolution GO.

Diferencias con v2:

- Los bots v2 se guardan como bots de Evolution GO (`/bots/:id`), así que
  también aparecen en el manager. OpenAI solo admite `chatCompletion`; el
  disparador `advanced` y los ajustes por línea (`/…/settings`,
  `/…/ignoreJid`) no existen: esas opciones van en cada bot.
- WebSocket, RabbitMQ y NATS comparten la lista de eventos de la línea con el
  webhook.
- `syncFullHistory` siempre es `true`: al vincular se pide el historial
  completo.
- No hay plantillas de la API oficial de Meta ni caché en Redis. Las sesiones
  de WhatsApp viven en la memoria del proceso, así que varias réplicas no
  pueden compartir una línea aunque compartan caché: para repartir carga, cada
  réplica debe llevar sus propias líneas (`CLIENT_NAME` distinto).

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
