<h1 align="center">Enlace</h1>

<p align="center">
  <b>Una API de WhatsApp autoalojada con un gestor moderno, historial de chats, bots y compatibilidad con Evolution API v2.</b><br/>
  Código abierto · Escrito en Go · Basado en Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <b>Español</b> ·
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
  <img src="./docs/images/dashboard.png" alt="Panel de Enlace: líneas operativas, caídas, entrega de webhooks y disponibilidad por hora" width="100%" />
</p>

---

## Por qué Enlace

Operar muchos números de WhatsApp consiste sobre todo en saber **qué líneas están sanas, cuáles se cayeron y volvieron, y cuáles necesitan a una persona**. Enlace está construido en torno a eso:

- **Ve todas las líneas de un vistazo.** Las líneas se dividen en *conectadas*, *reconectadas* (se cayeron y volvieron solas), *reconectando* y *caídas*, con un mapa de disponibilidad hora por hora y un registro de cada corte.
- **Arréglalas desde un solo lugar.** Reconecta, vincula por QR o código de emparejamiento, cierra sesión, configura un proxy o cambia ajustes sin tocar la API.
- **Lee y responde chats.** Cada línea guarda sus chats, contactos y mensajes en PostgreSQL, y el gestor los muestra como una app de mensajería.
- **Automatiza respuestas.** Bots por línea con n8n, OpenAI, Typebot, Dify, Flowise o tu propio endpoint HTTP, además de Chatwoot bidireccional.
- **Conecta lo que ya usas.** Webhooks firmados con reintentos, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher y métricas de Prometheus.
- **Conserva tus integraciones de Evolution API v2.** Las rutas y los nombres de eventos v2 funcionan sobre el mismo servidor, así que n8n, Typebot y los scripts existentes siguen funcionando sin cambios.
- **En tu idioma.** El gestor habla inglés, español, portugués, francés, italiano, alemán, ruso, chino, japonés y coreano, en escritorio y en móvil.

## Capturas de pantalla

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Líneas agrupadas por estado" /> | <img src="./docs/images/line-detail.png" alt="Detalle de línea con caídas y mapa de conexión de 7 días" /> |
| **Líneas** agrupadas por estado, filtrables por tus propios grupos | **Detalle de línea**: caídas, corte más largo y los últimos 7 días hora por hora |
| <img src="./docs/images/chats.png" alt="Chats con historial" /> | <img src="./docs/images/chat-menu.png" alt="Acciones del chat" /> |
| **Chats** guardados por línea, con marcas de entrega y lectura | **Acciones del chat**: archivar, fijar, silenciar, marcar como no leído, mensajes temporales |
| <img src="./docs/images/bots.png" alt="Bots por línea" /> | <img src="./docs/images/webhook.png" alt="Ajustes del webhook" /> |
| **Bots** por línea, con sesiones y Chatwoot | **Webhooks**: eventos, formato v2, archivos en base64, firma y encabezados |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Panel en el móvil" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Chat en el móvil" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Líneas en japonés en el móvil" width="30%" />
</p>
<p align="center"><sub>El mismo gestor en un teléfono, aquí en inglés y japonés.</sub></p>

### Principios de diseño

- **Tranquilo por defecto, llamativo cuando importa.** Las líneas sanas pasan a segundo plano; las caídas y las líneas que necesitan a una persona aparecen primero, con una acción clara.
- **Sin decoración.** Sin bordes ni separadores, sin degradados: superficies suaves, espaciado generoso y color solo para el estado (verde conectada, azul reconectada, naranja reconectando, rojo caída).
- **Números legibles.** Los números de teléfono, tokens y horas usan una fuente monoespaciada; los conteos son grandes y sencillos.
- **La misma app en cada pantalla.** Una barra lateral en escritorio y una barra de pestañas inferior en móvil, con las mismas páginas y acciones.

---

## Inicio rápido

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

Crea ambas bases de datos una sola vez (`evogo_auth` y `evogo_users`), inícialo con `docker compose up -d` y abre **http://localhost:8080/manager**. Inicia sesión con la URL del servidor y tu `GLOBAL_API_KEY`, crea una línea y vincúlala con el código QR.

### Desarrollo local

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` ejecuta el gestor con Vite en el puerto 5173. Ejecuta `make help` para ver todos los comandos.

---

## Funcionalidades

### Líneas y salud de la conexión

- `GET /instance/overview` devuelve todas las líneas con su estado, las caídas de las últimas 24 h, el corte más largo y un mapa por hora. `GET /instance/history/:id` devuelve el registro de cortes.
- `POST /instance/reconnect/:id` intenta recuperar una línea caída sin desvincularla.
- Las líneas se inician al arrancar (`CONNECT_ON_STARTUP`, activado por defecto) y se pueden organizar en tus propios **grupos** (`/line-groups`).

### Webhooks

Cada línea tiene su URL y sus eventos (configurados con `/instance/connect`), además de sus propios ajustes en `GET/PUT /webhook/:instanceId`:

- **Encabezados adicionales**, **una URL por evento** (`…/message`, `…/send-message`) y **archivos en base64** (`"base64": true`).
- **Firma HMAC**: `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` con `X-Webhook-Timestamp`. Cada solicitud también lleva `X-Webhook-Event` y un `X-Webhook-Delivery` único.
- **Reintentos** con esperas crecientes (1 s, 2 s, 4 s… hasta 60 s, 8 intentos; los 4xx no se reintentan, salvo 408/425/429), un registro de entregas (`/webhook/:id/deliveries`) y un evento de prueba (`POST /webhook/:id/test`).
- **Formato Evolution API v2** por línea o para el webhook global (`WEBHOOK_FORMAT=evolution-api`).

### Historial de chats

Con `CHAT_STORE_ENABLED=true` (por defecto), cada línea guarda sus chats, contactos y mensajes: lo que recibe, lo que envía a través de la API y el historial que WhatsApp envía cuando se vincula la línea.

| Método | Ruta | Uso |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Chats, los fijados primero |
| GET | `/store/:id/messages?chat=<jid>&before=` | Mensajes, los más recientes primero |
| GET | `/store/:id/contacts?q=` | Contactos conocidos |
| POST | `/store/:id/read` | Marcar como leído y enviar confirmaciones de lectura |

Los mensajes se conservan durante `CHAT_STORE_RETENTION_DAYS` días (90 por defecto; 0 = para siempre).

### Acciones de chat, grupos y contactos

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` con `{"chat": "<jid>"}` o `{"number": "…"}`. `mute` acepta `duration` en segundos (8 h por defecto, `-1` = siempre).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (mensajes temporales), `POST /user/business` (perfil de empresa).
- `POST /send/contact` acepta `vcards: [...]` para enviar varios contactos en un solo mensaje.

### Bots

Varios bots por línea (`/bots/:instanceId`). En cada chat habla un solo bot a la vez: el primero, por prioridad, cuyo disparador coincida (cualquier mensaje, o una palabra clave con `contains`, `equals`, `startsWith`, `endsWith` o `regex`).

| Tipo | Qué hace |
|---|---|
| `http` | Envía cada mensaje como JSON a tu URL y responde con lo que devuelva |
| `n8n` | Lo mismo, contra un nodo Webhook de n8n |
| `openai` | Chat Completions con el historial reciente; transcribe notas de voz |
| `typebot` | Ejecuta un flujo publicado de Typebot |
| `dify` | Apps de chat, agente, workflow o generador de texto |
| `flowise` | Un chatflow de Flowise |

Opciones comunes: responder en grupos, pausar cuando responde una persona, retraso de escritura, agrupación de mensajes rápidos, división de respuestas largas, palabra clave de cierre, expiración y chats ignorados.

### Chatwoot

`PUT /chatwoot/:instanceId` con la URL, la cuenta y el token de tu Chatwoot. Enlace crea una bandeja de entrada de tipo API, replica cada chat de WhatsApp (con adjuntos) como una conversación y envía las respuestas de los agentes a través de la línea.

### Compatibilidad con Evolution API v2

Las rutas v2 conviven con las de Go. Las líneas se identifican por **nombre** y `apikey` acepta la clave global o el token de la línea.

| Grupo | Rutas |
|---|---|
| Instancia | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Mensajes | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Chat | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Grupos | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Ajustes | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` y `find`) |
| Bots | `typebot`, `openai` (con `creds`), `dify`, `flowise`, `n8n`, `evolutionBot` y `chatwoot` |

Los eventos se pueden enviar con los nombres y el envoltorio de v2 (`messages.upsert`, `connection.update`, …). Las diferencias con v2 se detallan en [docs/COMPAT.md](./docs/COMPAT.md).

### Destinos de eventos

Además de webhooks, RabbitMQ, NATS y WebSocket, los eventos de cada línea pueden ir a:

| Destino | Variables principales |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` o `PUSHER_HOST` |

Cada uno acepta `<DESTINATION>_EVENTS` (filtro) y `<DESTINATION>_FORMAT=evolution-api`.

### Métricas

`GET /metrics` expone métricas de Prometheus: líneas por estado, mensajes recibidos y enviados, llamadas, entregas de webhooks y tiempo de entrega, además de datos del proceso. Requiere la clave global (`apikey` o `Authorization: Bearer`) o autenticación básica con `METRICS_USER` / `METRICS_PASSWORD`.

---

## Configuración

| Variable | Descripción | Valor por defecto |
|---|---|---|
| `GLOBAL_API_KEY` | Clave de administración para la API y el gestor | **obligatoria** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | URLs de conexión a PostgreSQL | **obligatoria** |
| `SERVER_PORT` | Puerto HTTP | `8080` |
| `CLIENT_NAME` | Agrupa las líneas de este servidor (útil cuando varios servidores comparten una base de datos) | — |
| `CONNECT_ON_STARTUP` | Reconecta todas las líneas al arrancar | `true` |
| `DATABASE_SAVE_MESSAGES` | Guarda el estado de entrega de los mensajes (`true` o `false`) | **obligatoria** |
| `CHAT_STORE_ENABLED` | Guarda chats, contactos y mensajes | `true` |
| `WEBHOOK_URL` | Webhook global para todas las líneas | — |
| `WEBHOOK_FORMAT` | `evolution-api` para el formato de eventos v2 | — |
| `SERVER_URL` | URL pública, usada en los eventos v2 | — |
| `METRICS_ENABLED` | Activa o desactiva `/metrics` | `true` |

La lista completa, con RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher y los reintentos de webhooks, está en [.env.example](./.env.example).

## Documentación de la API

Swagger UI se sirve en `/swagger/index.html`. El gestor usa la misma API, así que todo lo que ves en él se puede automatizar.

## Estructura del proyecto

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Privacidad

Enlace no envía telemetría ni datos de uso a ninguna parte. Solo se comunica con WhatsApp y con los destinos que configures.

## Mantenerse sincronizado con Evolution Go

Las nuevas funcionalidades viven en sus propios paquetes y solo tocan los archivos de upstream mediante ganchos breves. [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) explica cómo integrar las nuevas versiones de Evolution Go.

## Contribuir

Los issues y pull requests son bienvenidos. Ejecuta `go test ./...` y `make manager-build` antes de abrir un pull request, y mantén completas las traducciones del gestor en los diez idiomas.

## Créditos y licencia

Enlace está basado en [Evolution Go](https://github.com/evolution-foundation/evolution-go) de Evolution Foundation y usa [whatsmeow](https://github.com/tulir/whatsmeow) de Tulir Asokan.

Licenciado bajo Apache 2.0 con las condiciones adicionales de [LICENSE](./LICENSE). "Evolution Go" es una marca comercial de Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)); este proyecto es independiente y no está afiliado ni respaldado por Evolution Foundation. Los avisos de terceros están en [NOTICE](./NOTICE).
