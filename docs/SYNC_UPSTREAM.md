# Sincronizar con Evolution GO (upstream)

Este fork añade sus funciones en paquetes propios (`pkg/chatstore`,
`pkg/webhooks`, `pkg/chatbot`, `pkg/chatwoot`, `pkg/compat`, `pkg/evoformat`,
`pkg/metrics`, `pkg/sinks`, `pkg/extras`, `pkg/connlog`, `pkg/overview`,
`pkg/linegroups`, `manager/`) y toca los archivos de upstream solo con ganchos
cortos, para que traer versiones nuevas sea barato.

## Procedimiento

```bash
git fetch https://github.com/evolution-foundation/evolution-go main
git switch -c sync/upstream-<versión>
git merge FETCH_HEAD
go build ./... && go test ./...
```

Al resolver conflictos, conservar los ganchos del fork listados abajo y
aceptar el resto de cambios de upstream. `pkg/core/c0.go` (licencia) fue
eliminado: si upstream lo cambia, se mantiene borrado.

## Dónde toca el fork a upstream

| Archivo | Qué añade el fork |
|---|---|
| `cmd/evolution-go/main.go` | Migraciones y registro de los paquetes nuevos; `/ws` con token de línea; `/metrics`; la capa `compat` envuelve el router |
| `pkg/whatsmeow/service/whatsmeow.go` | Ganchos en `myEventHandler` (`chatstore`, `chatbot`, `chatwoot`, `metrics`), `sinks.Publish` en `CallWebhook`, `ClientMapsMu`, `connlog`, base64 por línea, `sendToQueueOrWebhook` sin cortar en errores |
| `pkg/sendMessage/service/send_service.go` | Ganchos `ObserveSent` (`chatstore`, `chatwoot`, `metrics`), base64 por línea, varios contactos |
| `pkg/chat/service/chat_service.go`, `pkg/chat/handler/chat_handler.go` | Archivar, fijar, silenciar y `markunread` reescritos sobre un ayudante común |
| `pkg/events/websocket/websocket_producer.go` | Varias conexiones por línea y autenticación con token |
| `pkg/events/webhook/webhook_producer.go` | Delegación en `pkg/webhooks` |
| `pkg/instance/*` | `readStatus`, `AuthInstanceOrAdmin`, `instance_id` en mensajes, borrado de tablas del fork |
| `pkg/middleware/auth_middleware.go` | `AuthInstanceOrAdmin` |
| `pkg/routes/routes.go` | `/chat/markunread`; sin rutas de licencia |
| `pkg/utils/utils.go` | `email` y `url` en la vCard |
| `pkg/config/config.go` | `CONNECT_ON_STARTUP` por defecto y nombres de variables antiguos |
| `Dockerfile`, `Makefile` | Compilación del manager |

Para ver el detalle exacto frente a la base:

```bash
git diff 80d7c72 HEAD --stat -- . ':!manager'
```

(`80d7c72` es «sync: 0.7.3 from main», el punto donde se separó el fork.)
