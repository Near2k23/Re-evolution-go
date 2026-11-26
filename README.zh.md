<h1 align="center">Enlace</h1>

<p align="center">
  <b>自托管的 WhatsApp API，配备现代化的管理面板、聊天记录、机器人，并兼容 Evolution API v2。</b><br/>
  开源 · 使用 Go 编写 · 基于 Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.it.md">Italiano</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <b>中文</b> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Enlace 仪表盘：正常运行的线路、掉线、webhook 投递情况和每小时可用性" width="100%" />
</p>

---

## 为什么选择 Enlace

运营大量 WhatsApp 号码，关键在于知道**哪些线路状态良好，哪些掉线后又恢复了，哪些需要人工处理**。Enlace 正是围绕这一点构建的：

- **一眼看清每条线路。** 线路分为*已连接*、*已重连*（掉线后自行恢复）、*重连中*和*已断开*，并提供逐小时的可用性图以及每次中断的日志。
- **在一个地方修复。** 无需调用 API，即可重新连接、通过二维码或配对码绑定、注销、设置代理或修改设置。
- **阅读并回复聊天。** 每条线路的聊天、联系人和消息都保存在 PostgreSQL 中，管理面板以即时通讯应用的方式展示它们。
- **自动回复。** 每条线路可配置机器人，支持 n8n、OpenAI、Typebot、Dify、Flowise 或你自己的 HTTP 端点，另有双向 Chatwoot 集成。
- **接入你已在使用的工具。** 带签名和重试的 webhook、RabbitMQ、NATS、WebSocket、Amazon SQS、Kafka、Pusher 以及 Prometheus 指标。
- **保留你的 Evolution API v2 集成。** v2 路由和事件名称在同一服务器上运行，因此 n8n、Typebot 和现有脚本无需修改即可使用。
- **使用你的语言。** 管理面板支持英语、西班牙语、葡萄牙语、法语、意大利语、德语、俄语、中文、日语和韩语，桌面端和移动端均可使用。

## 截图

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="按状态分组的线路" /> | <img src="./docs/images/line-detail.png" alt="线路详情，包含掉线记录和 7 天连接图" /> |
| **线路**按状态分组，可按你自定义的分组筛选 | **线路详情**：掉线次数、最长中断时间以及最近 7 天的逐小时情况 |
| <img src="./docs/images/chats.png" alt="带历史记录的聊天" /> | <img src="./docs/images/chat-menu.png" alt="聊天操作" /> |
| **聊天**按线路存储，带有送达和已读标记 | **聊天操作**：归档、置顶、静音、标记为未读、限时消息 |
| <img src="./docs/images/bots.png" alt="每条线路的机器人" /> | <img src="./docs/images/webhook.png" alt="Webhook 设置" /> |
| 每条线路的**机器人**，包含会话和 Chatwoot | **Webhook**：事件、v2 格式、base64 文件、签名和请求头 |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="移动端仪表盘" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="移动端聊天" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="移动端日语界面的线路" width="30%" />
</p>
<p align="center"><sub>同一个管理面板在手机上的样子，此处为英语和日语界面。</sub></p>

### 设计原则

- **平时安静，关键时醒目。** 状态良好的线路退居背景；掉线和需要人工处理的线路排在最前，并附带一个明确的操作。
- **没有多余装饰。** 不用边框或分隔线，不用渐变：柔和的界面、宽松的间距，颜色只用于表示状态（绿色已连接、蓝色已重连、橙色重连中、红色已断开）。
- **数字清晰易读。** 电话号码、令牌和时间使用等宽字体；计数大而简洁。
- **每种屏幕上都是同一个应用。** 桌面端使用侧边栏，移动端使用底部标签栏，页面和操作完全相同。

---

## 快速开始

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

先创建两个数据库（`evogo_auth` 和 `evogo_users`，只需一次），用 `docker compose up -d` 启动，然后打开 **http://localhost:8080/manager**。使用服务器 URL 和你的 `GLOBAL_API_KEY` 登录，创建一条线路并用二维码绑定。

### 本地开发

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` 会在 5173 端口用 Vite 运行管理面板。运行 `make help` 可查看所有命令。

---

## 功能

### 线路与连接健康状况

- `GET /instance/overview` 返回每条线路的状态、最近 24 小时的掉线次数、最长中断时间和逐小时图。`GET /instance/history/:id` 返回中断日志。
- `POST /instance/reconnect/:id` 尝试在不解除绑定的情况下恢复掉线的线路。
- 线路在启动时自动连接（`CONNECT_ON_STARTUP`，默认开启），并可以整理到你自定义的**分组**中（`/line-groups`）。

### Webhook

每条线路都有自己的 URL 和事件（通过 `/instance/connect` 设置），另外还有在 `GET/PUT /webhook/:instanceId` 中的独立设置：

- **额外请求头**、**每个事件单独的 URL**（`…/message`、`…/send-message`）以及 **base64 文件**（`"base64": true`）。
- **HMAC 签名**：`X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))`，并附带 `X-Webhook-Timestamp`。每个请求还带有 `X-Webhook-Event` 和唯一的 `X-Webhook-Delivery`。
- **重试**，等待时间逐步增加（1 秒、2 秒、4 秒……最长 60 秒，共 8 次尝试；4xx 不重试，408/425/429 除外），提供投递日志（`/webhook/:id/deliveries`）和测试事件（`POST /webhook/:id/test`）。
- **Evolution API v2 格式**，可按线路设置，也可用于全局 webhook（`WEBHOOK_FORMAT=evolution-api`）。

### 聊天记录

启用 `CHAT_STORE_ENABLED=true`（默认）后，每条线路都会存储其聊天、联系人和消息：包括收到的消息、通过 API 发送的消息，以及线路绑定时 WhatsApp 发送过来的历史记录。

| 方法 | 路由 | 用途 |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | 聊天，置顶的排在前面 |
| GET | `/store/:id/messages?chat=<jid>&before=` | 消息，最新的排在前面 |
| GET | `/store/:id/contacts?q=` | 已知联系人 |
| POST | `/store/:id/read` | 标记为已读并发送已读回执 |

消息保留 `CHAT_STORE_RETENTION_DAYS` 天（默认 90 天；0 = 永久保留）。

### 聊天操作、群组和联系人

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread`，参数为 `{"chat": "<jid>"}` 或 `{"number": "…"}`。`mute` 接受以秒为单位的 `duration`（默认 8 小时，`-1` = 永久）。
- `POST /group/inviteinfo`、`POST /group/ephemeral`（限时消息）、`POST /user/business`（商家资料）。
- `POST /send/contact` 接受 `vcards: [...]`，可在一条消息中发送多个联系人。

### 机器人

每条线路可配置多个机器人（`/bots/:instanceId`）。在每个聊天中，同一时间只有一个机器人对话：按优先级排序，第一个触发条件匹配的机器人生效（任意消息，或使用 `contains`、`equals`、`startsWith`、`endsWith` 或 `regex` 匹配的关键词）。

| 类型 | 作用 |
|---|---|
| `http` | 将每条消息以 JSON 形式 POST 到你的 URL，并用其返回内容回复 |
| `n8n` | 同上，目标为 n8n Webhook 节点 |
| `openai` | 结合最近历史记录调用 Chat Completions；可转写语音消息 |
| `typebot` | 运行已发布的 Typebot 流程 |
| `dify` | 聊天、智能体、工作流或文本生成类应用 |
| `flowise` | Flowise chatflow |

通用选项：在群组中回复、有人工回复时暂停、输入延迟、合并连续的快速消息、拆分长回复、结束关键词、过期时间以及忽略的聊天。

### Chatwoot

使用你的 Chatwoot URL、账户和令牌调用 `PUT /chatwoot/:instanceId`。Enlace 会创建一个 API 收件箱，将每个 WhatsApp 聊天（包括附件）镜像为一个会话，并通过该线路发送客服的回复。

### Evolution API v2 兼容性

v2 路由与 Go 路由并存。线路通过**名称**来指定，`apikey` 接受全局密钥或线路令牌。

| 分组 | 路由 |
|---|---|
| 实例 | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| 消息 | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| 聊天 | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| 群组 | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| 设置 | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats`（`set` 和 `find`） |
| 机器人 | `typebot`, `openai`（含 `creds`）, `dify`, `flowise`, `n8n`, `evolutionBot`，以及 `chatwoot` |

事件可以使用 v2 的名称和封装格式发送（`messages.upsert`、`connection.update` 等）。与 v2 的差异列在 [docs/COMPAT.md](./docs/COMPAT.md) 中。

### 事件目的地

除了 webhook、RabbitMQ、NATS 和 WebSocket 之外，每条线路的事件还可以发送到：

| 目的地 | 主要变量 |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` 或 `PUSHER_HOST` |

每个目的地都支持 `<DESTINATION>_EVENTS`（过滤）和 `<DESTINATION>_FORMAT=evolution-api`。

### 指标

`GET /metrics` 提供 Prometheus 指标：按状态统计的线路、收发的消息、通话、webhook 投递次数和投递耗时，以及进程数据。访问需要全局密钥（`apikey` 或 `Authorization: Bearer`），或使用 `METRICS_USER` / `METRICS_PASSWORD` 进行基本认证。

---

## 配置

| 变量 | 说明 | 默认值 |
|---|---|---|
| `GLOBAL_API_KEY` | API 和管理面板的管理员密钥 | **必填** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | PostgreSQL 连接 URL | **必填** |
| `SERVER_PORT` | HTTP 端口 | `8080` |
| `CLIENT_NAME` | 对本服务器的线路进行分组（多个服务器共用一个数据库时很有用） | — |
| `CONNECT_ON_STARTUP` | 启动时重新连接所有线路 | `true` |
| `DATABASE_SAVE_MESSAGES` | 存储消息投递状态（`true` 或 `false`） | **必填** |
| `CHAT_STORE_ENABLED` | 存储聊天、联系人和消息 | `true` |
| `WEBHOOK_URL` | 适用于所有线路的全局 webhook | — |
| `WEBHOOK_FORMAT` | 设为 `evolution-api` 以使用 v2 事件格式 | — |
| `SERVER_URL` | 公开 URL，用于 v2 事件 | — |
| `METRICS_ENABLED` | 开启或关闭 `/metrics` | `true` |

完整列表（包括 RabbitMQ、NATS、MinIO/S3、SQS、Kafka、Pusher 和 webhook 重试）见 [.env.example](./.env.example)。

## API 文档

Swagger UI 位于 `/swagger/index.html`。管理面板使用的是同一套 API，因此你在其中看到的一切都可以自动化。

## 项目结构

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## 隐私

Enlace 不会向任何地方发送遥测或使用数据。它只与 WhatsApp 以及你配置的目的地通信。

## 与 Evolution Go 保持同步

新功能位于独立的包中，仅通过简短的钩子触及上游文件。[docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) 说明了如何合并 Evolution Go 的新版本。

## 参与贡献

欢迎提交 issue 和 pull request。在提交 pull request 之前，请运行 `go test ./...` 和 `make manager-build`，并确保管理面板的十种语言翻译完整。

## 致谢与许可证

Enlace 基于 Evolution Foundation 的 [Evolution Go](https://github.com/evolution-foundation/evolution-go)，并使用了 Tulir Asokan 的 [whatsmeow](https://github.com/tulir/whatsmeow)。

本项目采用 Apache 2.0 许可证，并附有 [LICENSE](./LICENSE) 中的附加条件。"Evolution Go" 是 Evolution Foundation 的商标（[TRADEMARKS.md](./TRADEMARKS.md)）；本项目是独立项目，与 Evolution Foundation 无关联，也未获得其认可。第三方声明见 [NOTICE](./NOTICE)。
