<h1 align="center">Enlace</h1>

<p align="center">
  <b>Eine selbst gehostete WhatsApp-API mit modernem Manager, Chatverlauf, Bots und Kompatibilität zu Evolution API v2.</b><br/>
  Open Source · In Go geschrieben · Basiert auf Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.it.md">Italiano</a> ·
  <b>Deutsch</b> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Enlace-Dashboard: aktive Leitungen, Abbrüche, Webhook-Zustellung und stündliche Verfügbarkeit" width="100%" />
</p>

---

## Warum Enlace

Wer viele WhatsApp-Nummern betreibt, muss vor allem wissen, **welche Leitungen gesund sind, welche abgebrochen und wiedergekommen sind und welche einen Menschen brauchen**. Genau darauf ist Enlace ausgelegt:

- **Jede Leitung auf einen Blick.** Leitungen werden in *verbunden*, *wiederverbunden* (abgebrochen und von selbst zurückgekommen), *verbindet neu* und *getrennt* aufgeteilt, mit einer stündlichen Verfügbarkeitskarte und einem Protokoll jeder Unterbrechung.
- **Alles an einem Ort beheben.** Neu verbinden, per QR- oder Kopplungscode verknüpfen, abmelden, einen Proxy setzen oder Einstellungen ändern, ohne die API anzufassen.
- **Chats lesen und beantworten.** Jede Leitung speichert ihre Chats, Kontakte und Nachrichten in PostgreSQL, und der Manager zeigt sie wie eine Messaging-App an.
- **Antworten automatisieren.** Bots pro Leitung mit n8n, OpenAI, Typebot, Dify, Flowise oder deinem eigenen HTTP-Endpunkt, dazu bidirektionales Chatwoot.
- **Anbinden, was du bereits nutzt.** Signierte Webhooks mit Wiederholungen, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher und Prometheus-Metriken.
- **Behalte deine Evolution API v2-Integrationen.** Die v2-Routen und Eventnamen funktionieren auf demselben Server, sodass n8n, Typebot und bestehende Skripte unverändert laufen.
- **In deiner Sprache.** Der Manager spricht Englisch, Spanisch, Portugiesisch, Französisch, Italienisch, Deutsch, Russisch, Chinesisch, Japanisch und Koreanisch, auf Desktop und Mobilgerät.

## Screenshots

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Leitungen nach Status gruppiert" /> | <img src="./docs/images/line-detail.png" alt="Leitungsdetails mit Abbrüchen und 7-Tage-Verbindungskarte" /> |
| **Leitungen** nach Status gruppiert, filterbar nach deinen eigenen Gruppen | **Leitungsdetails**: Abbrüche, längste Unterbrechung und die letzten 7 Tage Stunde für Stunde |
| <img src="./docs/images/chats.png" alt="Chats mit Verlauf" /> | <img src="./docs/images/chat-menu.png" alt="Chat-Aktionen" /> |
| **Chats** pro Leitung gespeichert, mit Zustell- und Lesehäkchen | **Chat-Aktionen**: archivieren, anheften, stummschalten, als ungelesen markieren, selbstlöschende Nachrichten |
| <img src="./docs/images/bots.png" alt="Bots pro Leitung" /> | <img src="./docs/images/webhook.png" alt="Webhook-Einstellungen" /> |
| **Bots** pro Leitung, mit Sitzungen und Chatwoot | **Webhooks**: Events, v2-Format, Dateien in base64, Signatur und Header |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Dashboard auf dem Handy" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Chat auf dem Handy" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Leitungen auf Japanisch auf dem Handy" width="30%" />
</p>
<p align="center"><sub>Derselbe Manager auf einem Smartphone, hier auf Englisch und Japanisch.</sub></p>

### Designprinzipien

- **Standardmäßig ruhig, deutlich, wenn es darauf ankommt.** Gesunde Leitungen treten in den Hintergrund; Abbrüche und Leitungen, die einen Menschen brauchen, stehen vorne, mit einer klaren Aktion.
- **Keine Dekoration.** Keine Rahmen oder Trennlinien, keine Verläufe: weiche Flächen, großzügige Abstände und Farbe nur für den Status (grün verbunden, blau wiederverbunden, orange verbindet neu, rot getrennt).
- **Lesbare Zahlen.** Telefonnummern, Tokens und Uhrzeiten verwenden eine Monospace-Schrift; Zählwerte sind groß und schlicht.
- **Dieselbe App auf jedem Bildschirm.** Eine Seitenleiste auf dem Desktop und eine Tab-Leiste unten auf dem Handy, mit denselben Seiten und Aktionen.

---

## Schnellstart

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

Lege beide Datenbanken einmalig an (`evogo_auth` und `evogo_users`), starte mit `docker compose up -d` und öffne **http://localhost:8080/manager**. Melde dich mit der Server-URL und deinem `GLOBAL_API_KEY` an, erstelle eine Leitung und verknüpfe sie mit dem QR-Code.

### Lokale Entwicklung

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` startet den Manager mit Vite auf Port 5173. Mit `make help` siehst du alle Befehle.

---

## Funktionen

### Leitungen und Verbindungszustand

- `GET /instance/overview` liefert jede Leitung mit ihrem Status, den Abbrüchen der letzten 24 h, der längsten Unterbrechung und einer stündlichen Karte. `GET /instance/history/:id` liefert das Unterbrechungsprotokoll.
- `POST /instance/reconnect/:id` versucht, eine abgebrochene Leitung zurückzuholen, ohne die Verknüpfung aufzuheben.
- Leitungen starten beim Hochfahren (`CONNECT_ON_STARTUP`, standardmäßig aktiv) und lassen sich in eigenen **Gruppen** organisieren (`/line-groups`).

### Webhooks

Jede Leitung hat ihre URL und Events (gesetzt mit `/instance/connect`) sowie eigene Einstellungen unter `GET/PUT /webhook/:instanceId`:

- **Zusätzliche Header**, **eine URL pro Event** (`…/message`, `…/send-message`) und **Dateien in base64** (`"base64": true`).
- **HMAC-Signatur**: `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` mit `X-Webhook-Timestamp`. Jede Anfrage enthält außerdem `X-Webhook-Event` und einen eindeutigen `X-Webhook-Delivery`.
- **Wiederholungen** mit wachsenden Wartezeiten (1 s, 2 s, 4 s… bis 60 s, 8 Versuche; 4xx wird nicht wiederholt, außer 408/425/429), ein Zustellprotokoll (`/webhook/:id/deliveries`) und ein Test-Event (`POST /webhook/:id/test`).
- **Evolution API v2-Format** pro Leitung oder für den globalen Webhook (`WEBHOOK_FORMAT=evolution-api`).

### Chatverlauf

Mit `CHAT_STORE_ENABLED=true` (Standard) speichert jede Leitung ihre Chats, Kontakte und Nachrichten: was sie empfängt, was sie über die API sendet und den Verlauf, den WhatsApp beim Verknüpfen der Leitung schickt.

| Methode | Route | Zweck |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Chats, angeheftete zuerst |
| GET | `/store/:id/messages?chat=<jid>&before=` | Nachrichten, neueste zuerst |
| GET | `/store/:id/contacts?q=` | Bekannte Kontakte |
| POST | `/store/:id/read` | Als gelesen markieren und Lesebestätigungen senden |

Nachrichten werden `CHAT_STORE_RETENTION_DAYS` Tage aufbewahrt (standardmäßig 90; 0 = für immer).

### Chat-Aktionen, Gruppen und Kontakte

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` mit `{"chat": "<jid>"}` oder `{"number": "…"}`. `mute` akzeptiert `duration` in Sekunden (standardmäßig 8 h, `-1` = dauerhaft).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (selbstlöschende Nachrichten), `POST /user/business` (Unternehmensprofil).
- `POST /send/contact` akzeptiert `vcards: [...]`, um mehrere Kontakte in einer Nachricht zu senden.

### Bots

Mehrere Bots pro Leitung (`/bots/:instanceId`). In jedem Chat spricht jeweils nur ein Bot: der erste nach Priorität, dessen Auslöser passt (jede Nachricht oder ein Schlüsselwort mit `contains`, `equals`, `startsWith`, `endsWith` oder `regex`).

| Typ | Was er macht |
|---|---|
| `http` | Sendet jede Nachricht als JSON an deine URL und antwortet mit dem, was zurückkommt |
| `n8n` | Dasselbe, gegen einen n8n-Webhook-Knoten |
| `openai` | Chat Completions mit dem jüngsten Verlauf; transkribiert Sprachnachrichten |
| `typebot` | Führt einen veröffentlichten Typebot-Flow aus |
| `dify` | Chat-, Agent-, Workflow- oder Textgenerator-Apps |
| `flowise` | Ein Flowise-Chatflow |

Gemeinsame Optionen: in Gruppen antworten, pausieren, wenn ein Mensch antwortet, Tippverzögerung, Bündeln schneller Nachrichten, Aufteilen langer Antworten, Schlüsselwort zum Beenden, Ablaufzeit und ignorierte Chats.

### Chatwoot

`PUT /chatwoot/:instanceId` mit deiner Chatwoot-URL, deinem Konto und deinem Token. Enlace erstellt eine API-Inbox, spiegelt jeden WhatsApp-Chat (mit Anhängen) als Konversation und sendet die Antworten der Agenten über die Leitung.

### Kompatibilität mit Evolution API v2

Die v2-Routen liegen neben den Go-Routen. Leitungen werden über ihren **Namen** angesprochen, und `apikey` akzeptiert den globalen Schlüssel oder den Token der Leitung.

| Gruppe | Routen |
|---|---|
| Instanz | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Nachrichten | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Chat | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Gruppen | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Einstellungen | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` und `find`) |
| Bots | `typebot`, `openai` (mit `creds`), `dify`, `flowise`, `n8n`, `evolutionBot` und `chatwoot` |

Events können mit v2-Namen und -Envelope gesendet werden (`messages.upsert`, `connection.update`, …). Die Unterschiede zu v2 sind in [docs/COMPAT.md](./docs/COMPAT.md) aufgeführt.

### Event-Ziele

Neben Webhooks, RabbitMQ, NATS und WebSocket können die Events jeder Leitung gesendet werden an:

| Ziel | Wichtigste Variablen |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` oder `PUSHER_HOST` |

Jedes Ziel akzeptiert `<DESTINATION>_EVENTS` (Filter) und `<DESTINATION>_FORMAT=evolution-api`.

### Metriken

`GET /metrics` stellt Prometheus-Metriken bereit: Leitungen nach Status, empfangene und gesendete Nachrichten, Anrufe, Webhook-Zustellungen und Zustellzeit sowie Prozessdaten. Erforderlich ist der globale Schlüssel (`apikey` oder `Authorization: Bearer`) oder Basic Auth mit `METRICS_USER` / `METRICS_PASSWORD`.

---

## Konfiguration

| Variable | Beschreibung | Standard |
|---|---|---|
| `GLOBAL_API_KEY` | Admin-Schlüssel für die API und den Manager | **erforderlich** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | PostgreSQL-Verbindungs-URLs | **erforderlich** |
| `SERVER_PORT` | HTTP-Port | `8080` |
| `CLIENT_NAME` | Gruppiert die Leitungen dieses Servers (nützlich, wenn sich mehrere Server eine Datenbank teilen) | — |
| `CONNECT_ON_STARTUP` | Verbindet beim Start jede Leitung neu | `true` |
| `DATABASE_SAVE_MESSAGES` | Speichert den Zustellstatus von Nachrichten (`true` oder `false`) | **erforderlich** |
| `CHAT_STORE_ENABLED` | Speichert Chats, Kontakte und Nachrichten | `true` |
| `WEBHOOK_URL` | Globaler Webhook für jede Leitung | — |
| `WEBHOOK_FORMAT` | `evolution-api` für das v2-Eventformat | — |
| `SERVER_URL` | Öffentliche URL, wird in v2-Events verwendet | — |
| `METRICS_ENABLED` | Schaltet `/metrics` ein oder aus | `true` |

Die vollständige Liste, mit RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher und Webhook-Wiederholungen, findest du in [.env.example](./.env.example).

## API-Dokumentation

Swagger UI wird unter `/swagger/index.html` bereitgestellt. Der Manager nutzt dieselbe API, sodass sich alles, was du darin siehst, automatisieren lässt.

## Projektstruktur

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Datenschutz

Enlace sendet keine Telemetrie oder Nutzungsdaten irgendwohin. Es kommuniziert nur mit WhatsApp und mit den Zielen, die du konfigurierst.

## Synchron mit Evolution Go bleiben

Neue Funktionen liegen in eigenen Paketen und berühren Upstream-Dateien nur über kurze Hooks. [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) erklärt, wie neue Evolution Go-Releases eingepflegt werden.

## Mitwirken

Issues und Pull Requests sind willkommen. Bitte führe `go test ./...` und `make manager-build` aus, bevor du einen Pull Request öffnest, und halte die Übersetzungen des Managers in allen zehn Sprachen vollständig.

## Danksagung und Lizenz

Enlace basiert auf [Evolution Go](https://github.com/evolution-foundation/evolution-go) von Evolution Foundation und verwendet [whatsmeow](https://github.com/tulir/whatsmeow) von Tulir Asokan.

Lizenziert unter Apache 2.0 mit den zusätzlichen Bedingungen in [LICENSE](./LICENSE). „Evolution Go“ ist eine Marke von Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)); dieses Projekt ist unabhängig und weder mit Evolution Foundation verbunden noch von ihr unterstützt. Hinweise zu Drittanbietern findest du in [NOTICE](./NOTICE).
