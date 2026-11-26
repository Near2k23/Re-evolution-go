<h1 align="center">Enlace</h1>

<p align="center">
  <b>현대적인 관리 화면, 채팅 기록, 봇, Evolution API v2 호환성을 갖춘 셀프 호스팅 WhatsApp API입니다.</b><br/>
  오픈 소스 · Go로 작성 · Evolution Go 기반
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
  <a href="./README.ja.md">日本語</a> ·
  <b>한국어</b>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Enlace 대시보드: 운영 중인 회선, 끊김, webhook 전송, 시간별 가용성" width="100%" />
</p>

---

## 왜 Enlace인가

여러 WhatsApp 번호를 운영할 때 핵심은 **어떤 회선이 정상인지, 어떤 회선이 끊겼다가 돌아왔는지, 어떤 회선에 사람의 손이 필요한지**를 아는 것입니다. Enlace는 바로 이를 중심으로 만들어졌습니다.

- **모든 회선을 한눈에 확인합니다.** 회선은 *연결됨*, *재연결됨*(끊겼다가 스스로 복구됨), *재연결 중*, *중단됨*으로 나뉘며, 시간별 가용성 지도와 모든 끊김의 기록을 제공합니다.
- **한곳에서 복구합니다.** API를 건드리지 않고도 재연결, QR 코드나 페어링 코드로 연결, 로그아웃, 프록시 설정, 설정 변경을 할 수 있습니다.
- **채팅을 읽고 답장합니다.** 각 회선의 채팅, 연락처, 메시지는 PostgreSQL에 저장되며, 관리 화면은 이를 메신저 앱처럼 보여 줍니다.
- **답장을 자동화합니다.** 회선마다 n8n, OpenAI, Typebot, Dify, Flowise 또는 직접 만든 HTTP 엔드포인트로 봇을 구성할 수 있으며, Chatwoot와의 양방향 연동도 지원합니다.
- **이미 사용 중인 도구와 연결합니다.** 서명과 재시도를 지원하는 webhook, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher, Prometheus 메트릭을 지원합니다.
- **Evolution API v2 연동을 그대로 유지합니다.** v2 라우트와 이벤트 이름이 같은 서버 위에서 동작하므로 n8n, Typebot, 기존 스크립트를 수정 없이 사용할 수 있습니다.
- **여러분의 언어로.** 관리 화면은 영어, 스페인어, 포르투갈어, 프랑스어, 이탈리아어, 독일어, 러시아어, 중국어, 일본어, 한국어를 지원하며, 데스크톱과 모바일에서 모두 사용할 수 있습니다.

## 스크린샷

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="상태별로 묶인 회선" /> | <img src="./docs/images/line-detail.png" alt="끊김 기록과 7일 연결 지도가 있는 회선 상세" /> |
| 상태별로 묶인 **회선**, 직접 만든 그룹으로 필터링 가능 | **회선 상세**: 끊김, 가장 긴 끊김, 최근 7일간의 시간별 상태 |
| <img src="./docs/images/chats.png" alt="기록이 있는 채팅" /> | <img src="./docs/images/chat-menu.png" alt="채팅 작업" /> |
| 회선별로 저장되는 **채팅**, 전송 및 읽음 표시 포함 | **채팅 작업**: 보관, 고정, 알림 끄기, 읽지 않음으로 표시, 사라지는 메시지 |
| <img src="./docs/images/bots.png" alt="회선별 봇" /> | <img src="./docs/images/webhook.png" alt="Webhook 설정" /> |
| 회선별 **봇**, 세션 및 Chatwoot 포함 | **Webhook**: 이벤트, v2 형식, base64 파일, 서명과 헤더 |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="모바일 대시보드" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="모바일 채팅" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="모바일에서 일본어로 표시된 회선" width="30%" />
</p>
<p align="center"><sub>휴대폰에서 본 같은 관리 화면으로, 여기서는 영어와 일본어로 표시되어 있습니다.</sub></p>

### 디자인 원칙

- **평소에는 조용하게, 중요할 때는 확실하게.** 정상 회선은 배경으로 물러나고, 끊긴 회선과 사람의 손이 필요한 회선이 하나의 명확한 작업과 함께 맨 앞에 표시됩니다.
- **장식 없음.** 테두리나 구분선, 그라데이션을 쓰지 않습니다. 부드러운 면과 넉넉한 여백을 사용하고, 색상은 상태를 나타낼 때만 씁니다(초록 연결됨, 파랑 재연결됨, 주황 재연결 중, 빨강 중단됨).
- **읽기 쉬운 숫자.** 전화번호, 토큰, 시간에는 고정폭 글꼴을 사용하고, 개수는 크고 단순하게 표시합니다.
- **모든 화면에서 같은 앱.** 데스크톱에서는 사이드바, 모바일에서는 하단 탭 바를 사용하며 페이지와 작업은 동일합니다.

---

## 빠른 시작

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

두 데이터베이스(`evogo_auth`와 `evogo_users`)를 한 번 생성하고, `docker compose up -d`로 실행한 뒤 **http://localhost:8080/manager**를 엽니다. 서버 URL과 `GLOBAL_API_KEY`로 로그인하고, 회선을 만든 다음 QR 코드로 연결합니다.

### 로컬 개발

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev`는 Vite로 5173 포트에서 관리 화면을 실행합니다. 모든 명령은 `make help`로 확인할 수 있습니다.

---

## 기능

### 회선과 연결 상태

- `GET /instance/overview`는 모든 회선의 상태, 최근 24시간의 끊김 횟수, 가장 긴 끊김, 시간별 지도를 반환합니다. `GET /instance/history/:id`는 끊김 기록을 반환합니다.
- `POST /instance/reconnect/:id`는 연결을 해제하지 않고 끊긴 회선을 복구하려고 시도합니다.
- 회선은 부팅 시 시작되며(`CONNECT_ON_STARTUP`, 기본값 켜짐), 직접 만든 **그룹**(`/line-groups`)으로 정리할 수 있습니다.

### Webhook

각 회선에는 URL과 이벤트(`/instance/connect`로 설정)가 있으며, `GET/PUT /webhook/:instanceId`에서 회선별 설정도 할 수 있습니다.

- **추가 헤더**, **이벤트별 URL**(`…/message`, `…/send-message`), **base64 파일**(`"base64": true`).
- **HMAC 서명**: `X-Webhook-Timestamp`와 함께 `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))`. 모든 요청에는 `X-Webhook-Event`와 고유한 `X-Webhook-Delivery`도 포함됩니다.
- **재시도**는 대기 시간을 점점 늘려 가며 수행합니다(1초, 2초, 4초… 최대 60초, 8회 시도. 4xx는 408/425/429를 제외하고 재시도하지 않습니다). 전송 로그(`/webhook/:id/deliveries`)와 테스트 이벤트(`POST /webhook/:id/test`)도 제공합니다.
- **Evolution API v2 형식**을 회선별로 또는 전역 webhook에 적용할 수 있습니다(`WEBHOOK_FORMAT=evolution-api`).

### 채팅 기록

`CHAT_STORE_ENABLED=true`(기본값)이면 각 회선은 채팅, 연락처, 메시지를 저장합니다. 받은 메시지, API를 통해 보낸 메시지, 그리고 회선을 연결할 때 WhatsApp이 보내 주는 기록이 대상입니다.

| 메서드 | 라우트 | 용도 |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | 채팅, 고정된 채팅 우선 |
| GET | `/store/:id/messages?chat=<jid>&before=` | 메시지, 최신순 |
| GET | `/store/:id/contacts?q=` | 알려진 연락처 |
| POST | `/store/:id/read` | 읽음으로 표시하고 읽음 확인 전송 |

메시지는 `CHAT_STORE_RETENTION_DAYS`일 동안 보관됩니다(기본값 90일, 0 = 영구 보관).

### 채팅 작업, 그룹, 연락처

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread`에 `{"chat": "<jid>"}` 또는 `{"number": "…"}`를 전달합니다. `mute`는 초 단위의 `duration`을 받습니다(기본값 8시간, `-1` = 항상).
- `POST /group/inviteinfo`, `POST /group/ephemeral`(사라지는 메시지), `POST /user/business`(비즈니스 프로필).
- `POST /send/contact`는 `vcards: [...]`를 받아 하나의 메시지로 여러 연락처를 보낼 수 있습니다.

### 봇

회선마다 여러 봇을 둘 수 있습니다(`/bots/:instanceId`). 각 채팅에서는 한 번에 하나의 봇만 대화하며, 우선순위에 따라 트리거가 일치하는 첫 번째 봇이 응답합니다(모든 메시지, 또는 `contains`, `equals`, `startsWith`, `endsWith`, `regex`로 지정한 키워드).

| 유형 | 동작 |
|---|---|
| `http` | 각 메시지를 JSON으로 지정한 URL에 POST하고, 반환된 내용으로 답장합니다 |
| `n8n` | 위와 같지만 n8n Webhook 노드를 대상으로 합니다 |
| `openai` | 최근 기록을 포함한 Chat Completions. 음성 메시지를 텍스트로 변환합니다 |
| `typebot` | 게시된 Typebot 플로를 실행합니다 |
| `dify` | 채팅, 에이전트, 워크플로, 텍스트 생성 앱 |
| `flowise` | Flowise chatflow |

공통 옵션: 그룹에서 답장, 사람이 답하면 일시 정지, 입력 지연, 빠르게 연달아 온 메시지 묶기, 긴 답장 나누기, 종료 키워드, 만료 시간, 무시할 채팅.

### Chatwoot

Chatwoot URL, 계정, 토큰으로 `PUT /chatwoot/:instanceId`를 호출합니다. Enlace는 API 받은편지함을 만들고, 모든 WhatsApp 채팅을(첨부 파일 포함) 대화로 미러링하며, 상담원의 답장을 해당 회선으로 보냅니다.

### Evolution API v2 호환성

v2 라우트는 Go 라우트와 나란히 존재합니다. 회선은 **이름**으로 지정하며, `apikey`에는 전역 키나 회선 토큰을 사용할 수 있습니다.

| 그룹 | 라우트 |
|---|---|
| 인스턴스 | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| 메시지 | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| 채팅 | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| 그룹 | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| 설정 | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats`(`set`과 `find`) |
| 봇 | `typebot`, `openai`(`creds` 포함), `dify`, `flowise`, `n8n`, `evolutionBot`, 그리고 `chatwoot` |

이벤트는 v2 이름과 엔벨로프로 보낼 수 있습니다(`messages.upsert`, `connection.update` 등). v2와의 차이점은 [docs/COMPAT.md](./docs/COMPAT.md)에 정리되어 있습니다.

### 이벤트 대상

webhook, RabbitMQ, NATS, WebSocket 외에도 모든 회선의 이벤트를 다음 대상으로 보낼 수 있습니다.

| 대상 | 주요 변수 |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` 또는 `PUSHER_HOST` |

각 대상은 `<DESTINATION>_EVENTS`(필터)와 `<DESTINATION>_FORMAT=evolution-api`를 지원합니다.

### 메트릭

`GET /metrics`는 Prometheus 메트릭을 제공합니다. 상태별 회선, 받은 메시지와 보낸 메시지, 통화, webhook 전송 횟수와 전송 시간, 그리고 프로세스 데이터가 포함됩니다. 접근하려면 전역 키(`apikey` 또는 `Authorization: Bearer`)나 `METRICS_USER` / `METRICS_PASSWORD`를 사용한 기본 인증이 필요합니다.

---

## 설정

| 변수 | 설명 | 기본값 |
|---|---|---|
| `GLOBAL_API_KEY` | API와 관리 화면의 관리자 키 | **필수** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | PostgreSQL 연결 URL | **필수** |
| `SERVER_PORT` | HTTP 포트 | `8080` |
| `CLIENT_NAME` | 이 서버의 회선을 묶습니다(여러 서버가 하나의 데이터베이스를 공유할 때 유용) | — |
| `CONNECT_ON_STARTUP` | 부팅 시 모든 회선을 재연결 | `true` |
| `DATABASE_SAVE_MESSAGES` | 메시지 전송 상태 저장(`true` 또는 `false`) | **필수** |
| `CHAT_STORE_ENABLED` | 채팅, 연락처, 메시지 저장 | `true` |
| `WEBHOOK_URL` | 모든 회선에 적용되는 전역 webhook | — |
| `WEBHOOK_FORMAT` | v2 이벤트 형식을 쓰려면 `evolution-api` | — |
| `SERVER_URL` | 공개 URL, v2 이벤트에 사용 | — |
| `METRICS_ENABLED` | `/metrics` 켜기/끄기 | `true` |

RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher, webhook 재시도를 포함한 전체 목록은 [.env.example](./.env.example)에 있습니다.

## API 문서

Swagger UI는 `/swagger/index.html`에서 제공됩니다. 관리 화면도 같은 API를 사용하므로, 화면에서 보이는 모든 것을 자동화할 수 있습니다.

## 프로젝트 구조

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## 개인정보 보호

Enlace는 원격 측정이나 사용 데이터를 어디에도 보내지 않습니다. WhatsApp과 여러분이 설정한 대상하고만 통신합니다.

## Evolution Go와 동기화 유지

새 기능은 별도의 패키지에 있으며, 업스트림 파일은 짧은 훅을 통해서만 건드립니다. Evolution Go의 새 릴리스를 병합하는 방법은 [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md)에서 설명합니다.

## 기여하기

이슈와 pull request를 환영합니다. pull request를 열기 전에 `go test ./...`와 `make manager-build`를 실행하고, 관리 화면 번역을 10개 언어 모두 빠짐없이 유지해 주시기 바랍니다.

## 크레딧 및 라이선스

Enlace는 Evolution Foundation의 [Evolution Go](https://github.com/evolution-foundation/evolution-go)를 기반으로 하며, Tulir Asokan의 [whatsmeow](https://github.com/tulir/whatsmeow)를 사용합니다.

Apache 2.0 라이선스와 [LICENSE](./LICENSE)에 명시된 추가 조건에 따라 배포됩니다. "Evolution Go"는 Evolution Foundation의 상표입니다([TRADEMARKS.md](./TRADEMARKS.md)). 이 프로젝트는 독립적인 프로젝트이며 Evolution Foundation과 제휴 관계가 없고, Evolution Foundation의 보증을 받지 않았습니다. 서드파티 고지는 [NOTICE](./NOTICE)에 있습니다.
