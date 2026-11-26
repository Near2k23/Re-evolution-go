<h1 align="center">Enlace</h1>

<p align="center">
  <b>Un'API WhatsApp self-hosted con un manager moderno, cronologia delle chat, bot e compatibilità con Evolution API v2.</b><br/>
  Open source · Scritto in Go · Basato su Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.fr.md">Français</a> ·
  <b>Italiano</b> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Dashboard di Enlace: linee operative, cadute, consegna dei webhook e disponibilità oraria" width="100%" />
</p>

---

## Perché Enlace

Gestire molti numeri WhatsApp significa soprattutto sapere **quali linee sono in salute, quali sono cadute e tornate, e quali hanno bisogno di una persona**. Enlace è costruito attorno a questo:

- **Vedi ogni linea a colpo d'occhio.** Le linee sono suddivise in *connesse*, *riconnesse* (cadute e tornate da sole), *in riconnessione* e *disconnesse*, con una mappa di disponibilità ora per ora e un registro di ogni interruzione.
- **Risolvi tutto da un unico posto.** Riconnetti, collega tramite QR o codice di abbinamento, esci, imposta un proxy o modifica le impostazioni senza toccare l'API.
- **Leggi e rispondi alle chat.** Ogni linea conserva chat, contatti e messaggi in PostgreSQL, e il manager li mostra come un'app di messaggistica.
- **Automatizza le risposte.** Bot per linea con n8n, OpenAI, Typebot, Dify, Flowise o un tuo endpoint HTTP, più Chatwoot bidirezionale.
- **Collega ciò che usi già.** Webhook firmati con nuovi tentativi, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher e metriche Prometheus.
- **Mantieni le tue integrazioni con Evolution API v2.** Le route e i nomi degli eventi v2 funzionano sullo stesso server, quindi n8n, Typebot e gli script esistenti continuano a funzionare senza modifiche.
- **Nella tua lingua.** Il manager parla inglese, spagnolo, portoghese, francese, italiano, tedesco, russo, cinese, giapponese e coreano, su desktop e mobile.

## Screenshot

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Linee raggruppate per stato" /> | <img src="./docs/images/line-detail.png" alt="Dettaglio della linea con cadute e mappa di connessione a 7 giorni" /> |
| **Linee** raggruppate per stato, filtrabili per gruppi personalizzati | **Dettaglio della linea**: cadute, interruzione più lunga e gli ultimi 7 giorni ora per ora |
| <img src="./docs/images/chats.png" alt="Chat con cronologia" /> | <img src="./docs/images/chat-menu.png" alt="Azioni della chat" /> |
| **Chat** salvate per linea, con spunte di consegna e lettura | **Azioni della chat**: archivia, fissa, silenzia, segna come non letto, messaggi effimeri |
| <img src="./docs/images/bots.png" alt="Bot per linea" /> | <img src="./docs/images/webhook.png" alt="Impostazioni dei webhook" /> |
| **Bot** per linea, con sessioni e Chatwoot | **Webhook**: eventi, formato v2, file in base64, firma e header |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Dashboard su mobile" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Chat su mobile" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Linee in giapponese su mobile" width="30%" />
</p>
<p align="center"><sub>Lo stesso manager su un telefono, qui in inglese e giapponese.</sub></p>

### Principi di design

- **Calmo per impostazione predefinita, evidente quando serve.** Le linee in salute restano sullo sfondo; le cadute e le linee che richiedono una persona vengono prima, con un'unica azione chiara.
- **Niente decorazioni.** Niente bordi né separatori, niente gradienti: superfici morbide, spaziatura generosa e colore solo per lo stato (verde connessa, blu riconnessa, arancione in riconnessione, rosso disconnessa).
- **Numeri leggibili.** Numeri di telefono, token e orari usano un carattere monospaziato; i conteggi sono grandi e semplici.
- **La stessa app su ogni schermo.** Una barra laterale su desktop e una barra di schede in basso su mobile, con le stesse pagine e azioni.

---

## Avvio rapido

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

Crea una sola volta entrambi i database (`evogo_auth` e `evogo_users`), avvia con `docker compose up -d` e apri **http://localhost:8080/manager**. Accedi con l'URL del server e la tua `GLOBAL_API_KEY`, crea una linea e collegala con il codice QR.

### Sviluppo locale

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` avvia il manager con Vite sulla porta 5173. Esegui `make help` per vedere tutti i comandi.

---

## Funzionalità

### Linee e stato della connessione

- `GET /instance/overview` restituisce ogni linea con il suo stato, le cadute delle ultime 24 h, l'interruzione più lunga e una mappa oraria. `GET /instance/history/:id` restituisce il registro delle interruzioni.
- `POST /instance/reconnect/:id` prova a ripristinare una linea caduta senza scollegarla.
- Le linee si avviano all'accensione (`CONNECT_ON_STARTUP`, attivo per impostazione predefinita) e possono essere organizzate in **gruppi** personalizzati (`/line-groups`).

### Webhook

Ogni linea ha il suo URL e i suoi eventi (impostati con `/instance/connect`), oltre a impostazioni proprie in `GET/PUT /webhook/:instanceId`:

- **Header aggiuntivi**, **un URL per evento** (`…/message`, `…/send-message`) e **file in base64** (`"base64": true`).
- **Firma HMAC**: `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` con `X-Webhook-Timestamp`. Ogni richiesta include anche `X-Webhook-Event` e un `X-Webhook-Delivery` univoco.
- **Nuovi tentativi** con attese crescenti (1 s, 2 s, 4 s… fino a 60 s, 8 tentativi; i 4xx non vengono ritentati tranne 408/425/429), un registro delle consegne (`/webhook/:id/deliveries`) e un evento di test (`POST /webhook/:id/test`).
- **Formato Evolution API v2** per linea o per il webhook globale (`WEBHOOK_FORMAT=evolution-api`).

### Cronologia delle chat

Con `CHAT_STORE_ENABLED=true` (predefinito) ogni linea salva chat, contatti e messaggi: ciò che riceve, ciò che invia tramite l'API e la cronologia che WhatsApp invia quando la linea viene collegata.

| Metodo | Route | Uso |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Chat, prima quelle fissate |
| GET | `/store/:id/messages?chat=<jid>&before=` | Messaggi, prima i più recenti |
| GET | `/store/:id/contacts?q=` | Contatti conosciuti |
| POST | `/store/:id/read` | Segna come letto e invia le conferme di lettura |

I messaggi vengono conservati per `CHAT_STORE_RETENTION_DAYS` giorni (90 per impostazione predefinita; 0 = per sempre).

### Azioni della chat, gruppi e contatti

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` con `{"chat": "<jid>"}` o `{"number": "…"}`. `mute` accetta `duration` in secondi (8 h per impostazione predefinita, `-1` = sempre).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (messaggi effimeri), `POST /user/business` (profilo aziendale).
- `POST /send/contact` accetta `vcards: [...]` per inviare più contatti in un unico messaggio.

### Bot

Più bot per linea (`/bots/:instanceId`). In ogni chat parla un solo bot alla volta: il primo, per priorità, il cui trigger corrisponde (qualsiasi messaggio, oppure una parola chiave con `contains`, `equals`, `startsWith`, `endsWith` o `regex`).

| Tipo | Cosa fa |
|---|---|
| `http` | Invia ogni messaggio come JSON al tuo URL e risponde con ciò che restituisce |
| `n8n` | Lo stesso, verso un nodo Webhook di n8n |
| `openai` | Chat Completions con la cronologia recente; trascrive i messaggi vocali |
| `typebot` | Esegue un flusso Typebot pubblicato |
| `dify` | App di chat, agente, workflow o generazione di testo |
| `flowise` | Un chatflow di Flowise |

Opzioni comuni: rispondere nei gruppi, mettere in pausa quando risponde una persona, ritardo di digitazione, raggruppamento dei messaggi rapidi, suddivisione delle risposte lunghe, parola chiave di chiusura, scadenza e chat ignorate.

### Chatwoot

`PUT /chatwoot/:instanceId` con l'URL, l'account e il token del tuo Chatwoot. Enlace crea una inbox API, replica ogni chat WhatsApp (con allegati) come conversazione e invia le risposte degli agenti tramite la linea.

### Compatibilità con Evolution API v2

Le route v2 convivono con quelle Go. Le linee si identificano per **nome** e `apikey` accetta la chiave globale o il token della linea.

| Gruppo | Route |
|---|---|
| Istanza | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Messaggi | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Chat | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Gruppi | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Impostazioni | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` e `find`) |
| Bot | `typebot`, `openai` (con `creds`), `dify`, `flowise`, `n8n`, `evolutionBot` e `chatwoot` |

Gli eventi possono essere inviati con i nomi e l'envelope v2 (`messages.upsert`, `connection.update`, …). Le differenze rispetto alla v2 sono elencate in [docs/COMPAT.md](./docs/COMPAT.md).

### Destinazioni degli eventi

Oltre a webhook, RabbitMQ, NATS e WebSocket, gli eventi di ogni linea possono essere inviati a:

| Destinazione | Variabili principali |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` o `PUSHER_HOST` |

Ognuna accetta `<DESTINATION>_EVENTS` (filtro) e `<DESTINATION>_FORMAT=evolution-api`.

### Metriche

`GET /metrics` espone metriche Prometheus: linee per stato, messaggi ricevuti e inviati, chiamate, consegne dei webhook e tempo di consegna, oltre ai dati del processo. Richiede la chiave globale (`apikey` o `Authorization: Bearer`) oppure l'autenticazione basic con `METRICS_USER` / `METRICS_PASSWORD`.

---

## Configurazione

| Variabile | Descrizione | Predefinito |
|---|---|---|
| `GLOBAL_API_KEY` | Chiave di amministrazione per l'API e il manager | **obbligatoria** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | URL di connessione a PostgreSQL | **obbligatoria** |
| `SERVER_PORT` | Porta HTTP | `8080` |
| `CLIENT_NAME` | Raggruppa le linee di questo server (utile quando più server condividono un database) | — |
| `CONNECT_ON_STARTUP` | Riconnette ogni linea all'avvio | `true` |
| `DATABASE_SAVE_MESSAGES` | Salva lo stato di consegna dei messaggi (`true` o `false`) | **obbligatoria** |
| `CHAT_STORE_ENABLED` | Salva chat, contatti e messaggi | `true` |
| `WEBHOOK_URL` | Webhook globale per ogni linea | — |
| `WEBHOOK_FORMAT` | `evolution-api` per il formato degli eventi v2 | — |
| `SERVER_URL` | URL pubblico, usato negli eventi v2 | — |
| `METRICS_ENABLED` | Attiva o disattiva `/metrics` | `true` |

L'elenco completo, con RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher e i nuovi tentativi dei webhook, si trova in [.env.example](./.env.example).

## Documentazione dell'API

Swagger UI è disponibile su `/swagger/index.html`. Il manager usa la stessa API, quindi tutto ciò che vedi al suo interno può essere automatizzato.

## Struttura del progetto

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Privacy

Enlace non invia telemetria né dati di utilizzo da nessuna parte. Comunica solo con WhatsApp e con le destinazioni che configuri.

## Restare in sincronia con Evolution Go

Le nuove funzionalità vivono in pacchetti propri e toccano i file upstream solo tramite brevi hook. [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) spiega come integrare le nuove versioni di Evolution Go.

## Contribuire

Issue e pull request sono benvenute. Esegui `go test ./...` e `make manager-build` prima di aprire una pull request, e mantieni complete le traduzioni del manager nelle dieci lingue.

## Crediti e licenza

Enlace è basato su [Evolution Go](https://github.com/evolution-foundation/evolution-go) di Evolution Foundation e usa [whatsmeow](https://github.com/tulir/whatsmeow) di Tulir Asokan.

Distribuito con licenza Apache 2.0 con le condizioni aggiuntive indicate in [LICENSE](./LICENSE). "Evolution Go" è un marchio di Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)); questo progetto è indipendente e non è affiliato né approvato da Evolution Foundation. Le note di terze parti si trovano in [NOTICE](./NOTICE).
