# Evolution API v2 compatibility: differences

Enlace answers the Evolution API v2 routes by translating them to its own API.
These are the known differences:

- **Bots.** v2 bots are stored as Enlace bots (`/bots/:id`), so they also show
  up in the manager. OpenAI supports `chatCompletion` only (no assistants). The
  `advanced` trigger and per-instance bot settings (`/…/settings`,
  `/…/ignoreJid`) do not exist: set those options on each bot. `ignoreJids`
  accepts `@g.us` and `@s.whatsapp.net` to ignore every group or every private chat.
- **Event lists.** WebSocket, RabbitMQ and NATS share the line's event list with
  the webhook; v2 keeps one list per destination.
- **Events without a v2 equivalent** are not sent when the v2 format is on.
  Contact, group, label and history events keep the Enlace data shape inside
  the v2 envelope.
- **`syncFullHistory`** is always `true`: the full history is requested when a
  line is linked.
- **Not available:** Meta Cloud API templates, business catalog and Redis cache.
  WhatsApp sessions live in the memory of the process, so several replicas
  cannot share a line even with a shared cache; to spread the load, give each
  replica its own lines (a different `CLIENT_NAME`).
