<h1 align="center">Enlace</h1>

<p align="center">
  <b>モダンな管理画面、チャット履歴、ボット、Evolution API v2 互換性を備えたセルフホスト型 WhatsApp API です。</b><br/>
  オープンソース · Go で実装 · Evolution Go がベース
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.it.md">Italiano</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.zh.md">中文</a> ·
  <b>日本語</b> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Enlace のダッシュボード：稼働中の回線、切断、webhook の配信状況、時間ごとの稼働状況" width="100%" />
</p>

---

## Enlace を選ぶ理由

多数の WhatsApp 番号を運用するうえで大切なのは、**どの回線が正常か、どの回線が切断されて復帰したか、どの回線に人の対応が必要か**を把握することです。Enlace はその考え方を中心に作られています。

- **すべての回線をひと目で確認。** 回線は*接続中*、*再接続済み*（切断後に自動で復帰）、*再接続中*、*停止*に分類され、1 時間ごとの稼働マップとすべての切断の記録を表示します。
- **一か所で復旧。** API に触れることなく、再接続、QR コードやペアリングコードによるリンク、ログアウト、プロキシの設定、各種設定の変更ができます。
- **チャットを読んで返信。** 各回線のチャット、連絡先、メッセージは PostgreSQL に保存され、管理画面ではメッセージングアプリのように表示されます。
- **返信を自動化。** 回線ごとに n8n、OpenAI、Typebot、Dify、Flowise、または独自の HTTP エンドポイントを使ったボットを設定でき、Chatwoot との双方向連携にも対応しています。
- **既存のツールと接続。** 署名とリトライ付きの webhook、RabbitMQ、NATS、WebSocket、Amazon SQS、Kafka、Pusher、Prometheus メトリクスに対応しています。
- **Evolution API v2 の連携をそのまま利用。** v2 のルートとイベント名が同じサーバー上で動作するため、n8n、Typebot、既存のスクリプトを変更せずに使えます。
- **あなたの言語で。** 管理画面は英語、スペイン語、ポルトガル語、フランス語、イタリア語、ドイツ語、ロシア語、中国語、日本語、韓国語に対応し、デスクトップでもモバイルでも使えます。

## スクリーンショット

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="状態ごとにグループ化された回線" /> | <img src="./docs/images/line-detail.png" alt="切断と 7 日間の接続マップを含む回線の詳細" /> |
| 状態ごとにグループ化された**回線**。独自のグループで絞り込めます | **回線の詳細**：切断、最長の切断時間、直近 7 日間の 1 時間ごとの状況 |
| <img src="./docs/images/chats.png" alt="履歴付きのチャット" /> | <img src="./docs/images/chat-menu.png" alt="チャット操作" /> |
| 回線ごとに保存される**チャット**。配信済み・既読のチェックマーク付き | **チャット操作**：アーカイブ、ピン留め、ミュート、未読にする、消えるメッセージ |
| <img src="./docs/images/bots.png" alt="回線ごとのボット" /> | <img src="./docs/images/webhook.png" alt="Webhook の設定" /> |
| 回線ごとの**ボット**。セッションと Chatwoot 付き | **Webhook**：イベント、v2 形式、base64 のファイル、署名とヘッダー |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="モバイルでのダッシュボード" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="モバイルでのチャット" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="モバイルでの日本語表示の回線" width="30%" />
</p>
<p align="center"><sub>同じ管理画面をスマートフォンで表示した様子です（ここでは英語と日本語）。</sub></p>

### デザインの原則

- **普段は静かに、必要なときははっきりと。** 正常な回線は背景に溶け込み、切断や人の対応が必要な回線が先頭に表示され、明確なアクションが 1 つ示されます。
- **装飾なし。** 枠線や区切り線、グラデーションは使いません。柔らかな面、ゆとりのある余白、そして色は状態の表示にのみ使います（緑は接続中、青は再接続済み、オレンジは再接続中、赤は停止）。
- **読みやすい数字。** 電話番号、トークン、時刻には等幅フォントを使い、件数は大きくシンプルに表示します。
- **どの画面でも同じアプリ。** デスクトップではサイドバー、モバイルでは下部のタブバーを使い、ページと操作は共通です。

---

## クイックスタート

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

最初に 2 つのデータベース（`evogo_auth` と `evogo_users`）を作成し、`docker compose up -d` で起動して **http://localhost:8080/manager** を開きます。サーバーの URL と `GLOBAL_API_KEY` でサインインし、回線を作成して QR コードでリンクします。

### ローカル開発

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` は Vite を使ってポート 5173 で管理画面を起動します。すべてのコマンドは `make help` で確認できます。

---

## 機能

### 回線と接続の健全性

- `GET /instance/overview` は、すべての回線について状態、直近 24 時間の切断、最長の切断時間、1 時間ごとのマップを返します。`GET /instance/history/:id` は切断の記録を返します。
- `POST /instance/reconnect/:id` は、リンクを解除せずに切断された回線の復帰を試みます。
- 回線は起動時に開始され（`CONNECT_ON_STARTUP`、デフォルトで有効）、独自の**グループ**（`/line-groups`）で整理できます。

### Webhook

各回線には URL とイベント（`/instance/connect` で設定）があり、さらに `GET/PUT /webhook/:instanceId` で個別の設定ができます。

- **追加ヘッダー**、**イベントごとの URL**（`…/message`、`…/send-message`）、**base64 のファイル**（`"base64": true`）。
- **HMAC 署名**：`X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` と `X-Webhook-Timestamp`。すべてのリクエストには `X-Webhook-Event` と一意の `X-Webhook-Delivery` も付きます。
- **リトライ**は待ち時間を段階的に延ばして行います（1 秒、2 秒、4 秒…最大 60 秒、8 回まで。4xx は 408/425/429 を除きリトライしません）。配信ログ（`/webhook/:id/deliveries`）とテストイベント（`POST /webhook/:id/test`）もあります。
- **Evolution API v2 形式**を回線ごと、またはグローバル webhook に設定できます（`WEBHOOK_FORMAT=evolution-api`）。

### チャット履歴

`CHAT_STORE_ENABLED=true`（デフォルト）の場合、各回線はチャット、連絡先、メッセージを保存します。対象は受信したもの、API 経由で送信したもの、そして回線のリンク時に WhatsApp から送られてくる履歴です。

| メソッド | ルート | 用途 |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | チャット（ピン留めが先頭） |
| GET | `/store/:id/messages?chat=<jid>&before=` | メッセージ（新しい順） |
| GET | `/store/:id/contacts?q=` | 既知の連絡先 |
| POST | `/store/:id/read` | 既読にして既読通知を送信 |

メッセージは `CHAT_STORE_RETENTION_DAYS` 日間保持されます（デフォルトは 90 日、0 = 無期限）。

### チャット操作、グループ、連絡先

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` に `{"chat": "<jid>"}` または `{"number": "…"}` を指定します。`mute` は秒単位の `duration` を受け取ります（デフォルトは 8 時間、`-1` = 常時）。
- `POST /group/inviteinfo`、`POST /group/ephemeral`（消えるメッセージ）、`POST /user/business`（ビジネスプロフィール）。
- `POST /send/contact` は `vcards: [...]` を受け付け、1 つのメッセージで複数の連絡先を送信できます。

### ボット

1 つの回線に複数のボットを設定できます（`/bots/:instanceId`）。各チャットで応答するボットは一度に 1 つだけで、優先度順にトリガーが一致した最初のボットが応答します（任意のメッセージ、または `contains`、`equals`、`startsWith`、`endsWith`、`regex` によるキーワード）。

| タイプ | 内容 |
|---|---|
| `http` | 各メッセージを JSON として指定の URL に POST し、返された内容で返信します |
| `n8n` | 同様の処理を n8n の Webhook ノードに対して行います |
| `openai` | 直近の履歴を使った Chat Completions。ボイスメッセージを文字起こしします |
| `typebot` | 公開済みの Typebot フローを実行します |
| `dify` | チャット、エージェント、ワークフロー、テキスト生成アプリ |
| `flowise` | Flowise の chatflow |

共通オプション：グループでの返信、人が返信したときの一時停止、入力中の遅延、連続した短いメッセージのまとめ、長い返信の分割、終了キーワード、有効期限、無視するチャット。

### Chatwoot

Chatwoot の URL、アカウント、トークンを指定して `PUT /chatwoot/:instanceId` を呼び出します。Enlace は API 受信箱を作成し、すべての WhatsApp チャットを（添付ファイルも含めて）会話としてミラーリングし、エージェントの返信をその回線から送信します。

### Evolution API v2 互換性

v2 のルートは Go のルートと並んで動作します。回線は**名前**で指定し、`apikey` にはグローバルキーまたは回線のトークンを使用できます。

| グループ | ルート |
|---|---|
| インスタンス | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| メッセージ | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| チャット | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| グループ | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| 設定 | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats`（`set` と `find`） |
| ボット | `typebot`, `openai`（`creds` 付き）, `dify`, `flowise`, `n8n`, `evolutionBot`、および `chatwoot` |

イベントは v2 の名前とエンベロープで送信できます（`messages.upsert`、`connection.update` など）。v2 との違いは [docs/COMPAT.md](./docs/COMPAT.md) に記載しています。

### イベントの送信先

webhook、RabbitMQ、NATS、WebSocket に加えて、各回線のイベントは次の送信先にも送れます。

| 送信先 | 主な変数 |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` または `PUSHER_HOST` |

いずれも `<DESTINATION>_EVENTS`（フィルター）と `<DESTINATION>_FORMAT=evolution-api` を指定できます。

### メトリクス

`GET /metrics` は Prometheus メトリクスを公開します。状態別の回線数、送受信メッセージ、通話、webhook の配信数と配信時間、さらにプロセスのデータが含まれます。アクセスにはグローバルキー（`apikey` または `Authorization: Bearer`）、もしくは `METRICS_USER` / `METRICS_PASSWORD` による Basic 認証が必要です。

---

## 設定

| 変数 | 説明 | デフォルト |
|---|---|---|
| `GLOBAL_API_KEY` | API と管理画面の管理者キー | **必須** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | PostgreSQL の接続 URL | **必須** |
| `SERVER_PORT` | HTTP ポート | `8080` |
| `CLIENT_NAME` | このサーバーの回線をグループ化します（複数のサーバーでデータベースを共有する場合に便利） | — |
| `CONNECT_ON_STARTUP` | 起動時にすべての回線を再接続 | `true` |
| `DATABASE_SAVE_MESSAGES` | メッセージの配信ステータスを保存（`true` または `false`） | **必須** |
| `CHAT_STORE_ENABLED` | チャット、連絡先、メッセージを保存 | `true` |
| `WEBHOOK_URL` | すべての回線に適用されるグローバル webhook | — |
| `WEBHOOK_FORMAT` | v2 のイベント形式にするには `evolution-api` | — |
| `SERVER_URL` | 公開 URL（v2 のイベントで使用） | — |
| `METRICS_ENABLED` | `/metrics` の有効/無効 | `true` |

RabbitMQ、NATS、MinIO/S3、SQS、Kafka、Pusher、webhook のリトライを含む完全な一覧は [.env.example](./.env.example) にあります。

## API ドキュメント

Swagger UI は `/swagger/index.html` で提供されています。管理画面も同じ API を使っているため、画面上でできることはすべて自動化できます。

## プロジェクト構成

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## プライバシー

Enlace はテレメトリや利用データをどこにも送信しません。通信するのは WhatsApp と、あなたが設定した送信先だけです。

## Evolution Go との同期

新機能はそれぞれ独立したパッケージにあり、上流のファイルには短いフックを通じてのみ手を加えています。Evolution Go の新しいリリースを取り込む方法は [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) で説明しています。

## コントリビューション

issue や pull request を歓迎します。pull request を作成する前に `go test ./...` と `make manager-build` を実行し、管理画面の翻訳を 10 言語すべてで揃えてください。

## クレジットとライセンス

Enlace は Evolution Foundation による [Evolution Go](https://github.com/evolution-foundation/evolution-go) をベースにしており、Tulir Asokan による [whatsmeow](https://github.com/tulir/whatsmeow) を使用しています。

Apache 2.0 ライセンスのもとで提供され、[LICENSE](./LICENSE) に記載された追加条件が適用されます。「Evolution Go」は Evolution Foundation の商標です（[TRADEMARKS.md](./TRADEMARKS.md)）。本プロジェクトは独立したものであり、Evolution Foundation とは提携しておらず、同財団の承認も受けていません。サードパーティに関する表示は [NOTICE](./NOTICE) にあります。
