<h1 align="center">Enlace</h1>

<p align="center">
  <b>Самостоятельно размещаемый WhatsApp API с современным менеджером, историей чатов, ботами и совместимостью с Evolution API v2.</b><br/>
  Открытый исходный код · Написан на Go · Основан на Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.it.md">Italiano</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <b>Русский</b> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Панель Enlace: работающие линии, обрывы, доставка вебхуков и почасовая доступность" width="100%" />
</p>

---

## Почему Enlace

Когда у вас много номеров WhatsApp, главное — знать, **какие линии в порядке, какие отключались и вернулись, а каким нужен человек**. Enlace построен именно вокруг этого:

- **Все линии с первого взгляда.** Линии разделены на *подключённые*, *переподключённые* (отключились и вернулись сами), *переподключаются* и *отключены*, с почасовой картой доступности и журналом каждого обрыва.
- **Всё исправляется в одном месте.** Переподключайте, привязывайте по QR-коду или коду сопряжения, выходите из аккаунта, задавайте прокси или меняйте настройки, не обращаясь к API.
- **Читайте чаты и отвечайте на них.** Каждая линия хранит свои чаты, контакты и сообщения в PostgreSQL, а менеджер показывает их как мессенджер.
- **Автоматизируйте ответы.** Боты для каждой линии с n8n, OpenAI, Typebot, Dify, Flowise или вашим собственным HTTP-эндпоинтом, а также двусторонняя интеграция с Chatwoot.
- **Подключайте то, чем уже пользуетесь.** Подписанные вебхуки с повторными попытками, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher и метрики Prometheus.
- **Сохраните интеграции с Evolution API v2.** Маршруты и названия событий v2 работают на том же сервере, поэтому n8n, Typebot и существующие скрипты работают без изменений.
- **На вашем языке.** Менеджер доступен на английском, испанском, португальском, французском, итальянском, немецком, русском, китайском, японском и корейском, на компьютере и на телефоне.

## Скриншоты

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Линии, сгруппированные по состоянию" /> | <img src="./docs/images/line-detail.png" alt="Карточка линии с обрывами и картой подключения за 7 дней" /> |
| **Линии**, сгруппированные по состоянию, с фильтром по вашим собственным группам | **Карточка линии**: обрывы, самый долгий обрыв и последние 7 дней по часам |
| <img src="./docs/images/chats.png" alt="Чаты с историей" /> | <img src="./docs/images/chat-menu.png" alt="Действия с чатом" /> |
| **Чаты**, сохранённые для каждой линии, с отметками о доставке и прочтении | **Действия с чатом**: архивировать, закрепить, отключить звук, отметить как непрочитанный, исчезающие сообщения |
| <img src="./docs/images/bots.png" alt="Боты для каждой линии" /> | <img src="./docs/images/webhook.png" alt="Настройки вебхуков" /> |
| **Боты** для каждой линии, с сессиями и Chatwoot | **Вебхуки**: события, формат v2, файлы в base64, подпись и заголовки |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Панель на телефоне" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Чат на телефоне" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Линии на японском на телефоне" width="30%" />
</p>
<p align="center"><sub>Тот же менеджер на телефоне, здесь на английском и японском.</sub></p>

### Принципы дизайна

- **Спокойно по умолчанию, заметно, когда это важно.** Исправные линии уходят на второй план; обрывы и линии, которым нужен человек, идут первыми, с одним понятным действием.
- **Без украшений.** Без рамок и разделителей, без градиентов: мягкие поверхности, просторные отступы и цвет только для состояния (зелёный — подключена, синий — переподключена, оранжевый — переподключается, красный — отключена).
- **Читаемые числа.** Номера телефонов, токены и время набраны моноширинным шрифтом; счётчики крупные и простые.
- **Одно приложение на любом экране.** Боковая панель на компьютере и нижняя панель вкладок на телефоне, с теми же страницами и действиями.

---

## Быстрый старт

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

Один раз создайте обе базы данных (`evogo_auth` и `evogo_users`), запустите `docker compose up -d` и откройте **http://localhost:8080/manager**. Войдите, указав URL сервера и ваш `GLOBAL_API_KEY`, создайте линию и привяжите её по QR-коду.

### Локальная разработка

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` запускает менеджер с Vite на порту 5173. Выполните `make help`, чтобы увидеть все команды.

---

## Возможности

### Линии и состояние подключения

- `GET /instance/overview` возвращает все линии с их состоянием, обрывами за последние 24 ч, самым долгим обрывом и почасовой картой. `GET /instance/history/:id` возвращает журнал обрывов.
- `POST /instance/reconnect/:id` пытается вернуть отключившуюся линию, не отвязывая её.
- Линии запускаются при старте (`CONNECT_ON_STARTUP`, включено по умолчанию), и их можно распределять по собственным **группам** (`/line-groups`).

### Вебхуки

У каждой линии есть свой URL и события (задаются через `/instance/connect`), а также собственные настройки в `GET/PUT /webhook/:instanceId`:

- **Дополнительные заголовки**, **отдельный URL для каждого события** (`…/message`, `…/send-message`) и **файлы в base64** (`"base64": true`).
- **Подпись HMAC**: `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` вместе с `X-Webhook-Timestamp`. Каждый запрос также содержит `X-Webhook-Event` и уникальный `X-Webhook-Delivery`.
- **Повторные попытки** с растущими паузами (1 с, 2 с, 4 с… до 60 с, всего попыток: 8; ответы 4xx не повторяются, кроме 408/425/429), журнал доставки (`/webhook/:id/deliveries`) и тестовое событие (`POST /webhook/:id/test`).
- **Формат Evolution API v2** для отдельной линии или для глобального вебхука (`WEBHOOK_FORMAT=evolution-api`).

### История чатов

При `CHAT_STORE_ENABLED=true` (по умолчанию) каждая линия хранит свои чаты, контакты и сообщения: то, что она получает, то, что отправляет через API, и историю, которую WhatsApp присылает при привязке линии.

| Метод | Маршрут | Назначение |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Чаты, закреплённые первыми |
| GET | `/store/:id/messages?chat=<jid>&before=` | Сообщения, новые первыми |
| GET | `/store/:id/contacts?q=` | Известные контакты |
| POST | `/store/:id/read` | Отметить как прочитанное и отправить отчёты о прочтении |

Сообщения хранятся `CHAT_STORE_RETENTION_DAYS` дней (по умолчанию 90; 0 = бессрочно).

### Действия с чатами, группы и контакты

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` с `{"chat": "<jid>"}` или `{"number": "…"}`. `mute` принимает `duration` в секундах (по умолчанию 8 ч, `-1` = навсегда).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (исчезающие сообщения), `POST /user/business` (бизнес-профиль).
- `POST /send/contact` принимает `vcards: [...]`, чтобы отправить несколько контактов одним сообщением.

### Боты

Несколько ботов на линию (`/bots/:instanceId`). В каждом чате одновременно отвечает только один бот: первый по приоритету, чей триггер срабатывает (любое сообщение или ключевое слово с `contains`, `equals`, `startsWith`, `endsWith` или `regex`).

| Тип | Что делает |
|---|---|
| `http` | Отправляет каждое сообщение в виде JSON на ваш URL и отвечает тем, что он вернёт |
| `n8n` | То же самое, но через узел Webhook в n8n |
| `openai` | Chat Completions с недавней историей; расшифровывает голосовые сообщения |
| `typebot` | Запускает опубликованный сценарий Typebot |
| `dify` | Приложения типа чат, агент, workflow или генератор текста |
| `flowise` | Chatflow в Flowise |

Общие параметры: ответы в группах, пауза, когда отвечает человек, задержка набора текста, объединение быстрых сообщений, разбиение длинных ответов, ключевое слово для завершения, срок действия и игнорируемые чаты.

### Chatwoot

`PUT /chatwoot/:instanceId` с URL, аккаунтом и токеном вашего Chatwoot. Enlace создаёт API-инбокс, зеркалирует каждый чат WhatsApp (с вложениями) как разговор и отправляет ответы агентов через линию.

### Совместимость с Evolution API v2

Маршруты v2 работают рядом с маршрутами Go. Линии адресуются по **имени**, а `apikey` принимает глобальный ключ или токен линии.

| Группа | Маршруты |
|---|---|
| Инстанс | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Сообщения | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Чат | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Группы | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Настройки | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` и `find`) |
| Боты | `typebot`, `openai` (с `creds`), `dify`, `flowise`, `n8n`, `evolutionBot`, а также `chatwoot` |

События можно отправлять с названиями и оболочкой v2 (`messages.upsert`, `connection.update`, …). Отличия от v2 перечислены в [docs/COMPAT.md](./docs/COMPAT.md).

### Получатели событий

Помимо вебхуков, RabbitMQ, NATS и WebSocket, события каждой линии можно отправлять в:

| Получатель | Основные переменные |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` или `PUSHER_HOST` |

Каждый из них принимает `<DESTINATION>_EVENTS` (фильтр) и `<DESTINATION>_FORMAT=evolution-api`.

### Метрики

`GET /metrics` отдаёт метрики Prometheus: линии по состоянию, полученные и отправленные сообщения, звонки, доставки вебхуков и время доставки, а также данные процесса. Требуется глобальный ключ (`apikey` или `Authorization: Bearer`) либо basic-аутентификация с `METRICS_USER` / `METRICS_PASSWORD`.

---

## Настройка

| Переменная | Описание | По умолчанию |
|---|---|---|
| `GLOBAL_API_KEY` | Административный ключ для API и менеджера | **обязательно** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | URL подключения к PostgreSQL | **обязательно** |
| `SERVER_PORT` | HTTP-порт | `8080` |
| `CLIENT_NAME` | Группирует линии этого сервера (полезно, когда несколько серверов используют одну базу данных) | — |
| `CONNECT_ON_STARTUP` | Переподключать все линии при запуске | `true` |
| `DATABASE_SAVE_MESSAGES` | Сохранять статус доставки сообщений (`true` или `false`) | **обязательно** |
| `CHAT_STORE_ENABLED` | Сохранять чаты, контакты и сообщения | `true` |
| `WEBHOOK_URL` | Глобальный вебхук для всех линий | — |
| `WEBHOOK_FORMAT` | `evolution-api` для формата событий v2 | — |
| `SERVER_URL` | Публичный URL, используется в событиях v2 | — |
| `METRICS_ENABLED` | Включить или выключить `/metrics` | `true` |

Полный список, включая RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher и повторные попытки вебхуков, находится в [.env.example](./.env.example).

## Документация API

Swagger UI доступен по адресу `/swagger/index.html`. Менеджер использует тот же API, поэтому всё, что вы в нём видите, можно автоматизировать.

## Структура проекта

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Конфиденциальность

Enlace никуда не отправляет телеметрию или данные об использовании. Он обменивается данными только с WhatsApp и с теми получателями, которые вы настроили.

## Синхронизация с Evolution Go

Новые возможности живут в собственных пакетах и затрагивают исходные файлы upstream только через короткие хуки. В [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) описано, как вливать новые релизы Evolution Go.

## Участие в разработке

Issues и pull requests приветствуются. Пожалуйста, запустите `go test ./...` и `make manager-build` перед открытием pull request и поддерживайте полноту переводов менеджера на всех десяти языках.

## Благодарности и лицензия

Enlace основан на [Evolution Go](https://github.com/evolution-foundation/evolution-go) от Evolution Foundation и использует [whatsmeow](https://github.com/tulir/whatsmeow) от Tulir Asokan.

Распространяется по лицензии Apache 2.0 с дополнительными условиями, указанными в [LICENSE](./LICENSE). «Evolution Go» — товарный знак Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)); этот проект независим, не аффилирован с Evolution Foundation и не одобрен ею. Уведомления о сторонних компонентах находятся в [NOTICE](./NOTICE).
