<h1 align="center">Enlace</h1>

<p align="center">
  <b>Uma API de WhatsApp auto-hospedada com um gerenciador moderno, histórico de conversas, bots e compatibilidade com a Evolution API v2.</b><br/>
  Código aberto · Escrito em Go · Baseado no Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <b>Português</b> ·
  <a href="./README.fr.md">Français</a> ·
  <a href="./README.it.md">Italiano</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Painel do Enlace: linhas operacionais, quedas, entrega de webhooks e disponibilidade por hora" width="100%" />
</p>

---

## Por que o Enlace

Operar muitos números de WhatsApp é, acima de tudo, saber **quais linhas estão saudáveis, quais caíram e voltaram, e quais precisam de uma pessoa**. O Enlace foi construído em torno disso:

- **Veja todas as linhas de relance.** As linhas são divididas em *conectadas*, *reconectadas* (caíram e voltaram sozinhas), *reconectando* e *fora do ar*, com um mapa de disponibilidade hora a hora e um registro de cada queda.
- **Resolva tudo em um só lugar.** Reconecte, vincule por QR ou código de pareamento, desconecte, configure um proxy ou altere configurações sem mexer na API.
- **Leia e responda conversas.** Cada linha guarda suas conversas, contatos e mensagens no PostgreSQL, e o gerenciador as exibe como um app de mensagens.
- **Automatize respostas.** Bots por linha com n8n, OpenAI, Typebot, Dify, Flowise ou seu próprio endpoint HTTP, além de Chatwoot bidirecional.
- **Conecte o que você já usa.** Webhooks assinados com novas tentativas, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher e métricas do Prometheus.
- **Mantenha suas integrações da Evolution API v2.** As rotas e os nomes de eventos v2 funcionam sobre o mesmo servidor, então n8n, Typebot e scripts existentes continuam funcionando sem alterações.
- **No seu idioma.** O gerenciador fala inglês, espanhol, português, francês, italiano, alemão, russo, chinês, japonês e coreano, no desktop e no celular.

## Capturas de tela

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Linhas agrupadas por estado" /> | <img src="./docs/images/line-detail.png" alt="Detalhe da linha com quedas e mapa de conexão de 7 dias" /> |
| **Linhas** agrupadas por estado, filtráveis pelos seus próprios grupos | **Detalhe da linha**: quedas, maior interrupção e os últimos 7 dias hora a hora |
| <img src="./docs/images/chats.png" alt="Conversas com histórico" /> | <img src="./docs/images/chat-menu.png" alt="Ações da conversa" /> |
| **Conversas** armazenadas por linha, com confirmações de entrega e leitura | **Ações da conversa**: arquivar, fixar, silenciar, marcar como não lida, mensagens temporárias |
| <img src="./docs/images/bots.png" alt="Bots por linha" /> | <img src="./docs/images/webhook.png" alt="Configurações do webhook" /> |
| **Bots** por linha, com sessões e Chatwoot | **Webhooks**: eventos, formato v2, arquivos em base64, assinatura e cabeçalhos |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Painel no celular" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Conversa no celular" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Linhas em japonês no celular" width="30%" />
</p>
<p align="center"><sub>O mesmo gerenciador em um celular, aqui em inglês e japonês.</sub></p>

### Princípios de design

- **Discreto por padrão, chamativo quando importa.** Linhas saudáveis ficam em segundo plano; quedas e linhas que precisam de uma pessoa aparecem primeiro, com uma ação clara.
- **Sem decoração.** Sem bordas nem divisórias, sem gradientes: superfícies suaves, espaçamento generoso e cor apenas para o estado (verde conectada, azul reconectada, laranja reconectando, vermelho fora do ar).
- **Números legíveis.** Números de telefone, tokens e horários usam uma fonte monoespaçada; as contagens são grandes e simples.
- **O mesmo app em todas as telas.** Uma barra lateral no desktop e uma barra de abas inferior no celular, com as mesmas páginas e ações.

---

## Início rápido

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

Crie os dois bancos de dados uma única vez (`evogo_auth` e `evogo_users`), inicie com `docker compose up -d` e abra **http://localhost:8080/manager**. Entre com a URL do servidor e sua `GLOBAL_API_KEY`, crie uma linha e vincule-a com o código QR.

### Desenvolvimento local

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` executa o gerenciador com Vite na porta 5173. Execute `make help` para ver todos os comandos.

---

## Funcionalidades

### Linhas e saúde da conexão

- `GET /instance/overview` retorna todas as linhas com seu estado, as quedas das últimas 24 h, a maior interrupção e um mapa por hora. `GET /instance/history/:id` retorna o registro de quedas.
- `POST /instance/reconnect/:id` tenta recuperar uma linha caída sem desvinculá-la.
- As linhas iniciam junto com o servidor (`CONNECT_ON_STARTUP`, ativado por padrão) e podem ser organizadas nos seus próprios **grupos** (`/line-groups`).

### Webhooks

Cada linha tem sua URL e seus eventos (definidos com `/instance/connect`), além de suas próprias configurações em `GET/PUT /webhook/:instanceId`:

- **Cabeçalhos extras**, **uma URL por evento** (`…/message`, `…/send-message`) e **arquivos em base64** (`"base64": true`).
- **Assinatura HMAC**: `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` com `X-Webhook-Timestamp`. Cada requisição também leva `X-Webhook-Event` e um `X-Webhook-Delivery` único.
- **Novas tentativas** com esperas crescentes (1 s, 2 s, 4 s… até 60 s, 8 tentativas; respostas 4xx não são repetidas, exceto 408/425/429), um registro de entregas (`/webhook/:id/deliveries`) e um evento de teste (`POST /webhook/:id/test`).
- **Formato Evolution API v2** por linha ou para o webhook global (`WEBHOOK_FORMAT=evolution-api`).

### Histórico de conversas

Com `CHAT_STORE_ENABLED=true` (padrão), cada linha armazena suas conversas, contatos e mensagens: o que recebe, o que envia pela API e o histórico que o WhatsApp envia quando a linha é vinculada.

| Método | Rota | Uso |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Conversas, fixadas primeiro |
| GET | `/store/:id/messages?chat=<jid>&before=` | Mensagens, mais recentes primeiro |
| GET | `/store/:id/contacts?q=` | Contatos conhecidos |
| POST | `/store/:id/read` | Marcar como lida e enviar confirmações de leitura |

As mensagens são mantidas por `CHAT_STORE_RETENTION_DAYS` dias (90 por padrão; 0 = para sempre).

### Ações de conversa, grupos e contatos

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` com `{"chat": "<jid>"}` ou `{"number": "…"}`. `mute` aceita `duration` em segundos (8 h por padrão, `-1` = sempre).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (mensagens temporárias), `POST /user/business` (perfil comercial).
- `POST /send/contact` aceita `vcards: [...]` para enviar vários contatos em uma única mensagem.

### Bots

Vários bots por linha (`/bots/:instanceId`). Em cada conversa, apenas um bot fala por vez: o primeiro, por prioridade, cujo gatilho corresponder (qualquer mensagem, ou uma palavra-chave com `contains`, `equals`, `startsWith`, `endsWith` ou `regex`).

| Tipo | O que faz |
|---|---|
| `http` | Envia cada mensagem como JSON para sua URL e responde com o que ela retornar |
| `n8n` | O mesmo, para um nó Webhook do n8n |
| `openai` | Chat Completions com o histórico recente; transcreve mensagens de voz |
| `typebot` | Executa um fluxo publicado do Typebot |
| `dify` | Apps de chat, agente, workflow ou gerador de texto |
| `flowise` | Um chatflow do Flowise |

Opções comuns: responder em grupos, pausar quando uma pessoa responde, atraso de digitação, agrupamento de mensagens rápidas, divisão de respostas longas, palavra-chave de encerramento, expiração e conversas ignoradas.

### Chatwoot

`PUT /chatwoot/:instanceId` com a URL, a conta e o token do seu Chatwoot. O Enlace cria uma caixa de entrada do tipo API, espelha cada conversa do WhatsApp (com anexos) como uma conversa e envia as respostas dos agentes pela linha.

### Compatibilidade com a Evolution API v2

As rotas v2 ficam ao lado das rotas Go. As linhas são identificadas pelo **nome** e `apikey` aceita a chave global ou o token da linha.

| Grupo | Rotas |
|---|---|
| Instância | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Mensagens | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Conversa | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Grupos | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Configurações | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` e `find`) |
| Bots | `typebot`, `openai` (com `creds`), `dify`, `flowise`, `n8n`, `evolutionBot` e `chatwoot` |

Os eventos podem ser enviados com os nomes e o envelope da v2 (`messages.upsert`, `connection.update`, …). As diferenças em relação à v2 estão listadas em [docs/COMPAT.md](./docs/COMPAT.md).

### Destinos de eventos

Além de webhooks, RabbitMQ, NATS e WebSocket, os eventos de cada linha podem ir para:

| Destino | Variáveis principais |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` ou `PUSHER_HOST` |

Cada um aceita `<DESTINATION>_EVENTS` (filtro) e `<DESTINATION>_FORMAT=evolution-api`.

### Métricas

`GET /metrics` expõe métricas do Prometheus: linhas por estado, mensagens recebidas e enviadas, chamadas, entregas de webhooks e tempo de entrega, além de dados do processo. Requer a chave global (`apikey` ou `Authorization: Bearer`) ou autenticação básica com `METRICS_USER` / `METRICS_PASSWORD`.

---

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `GLOBAL_API_KEY` | Chave de administração da API e do gerenciador | **obrigatória** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | URLs de conexão com o PostgreSQL | **obrigatória** |
| `SERVER_PORT` | Porta HTTP | `8080` |
| `CLIENT_NAME` | Agrupa as linhas deste servidor (útil quando vários servidores compartilham um banco de dados) | — |
| `CONNECT_ON_STARTUP` | Reconecta todas as linhas ao iniciar | `true` |
| `DATABASE_SAVE_MESSAGES` | Armazena o status de entrega das mensagens (`true` ou `false`) | **obrigatória** |
| `CHAT_STORE_ENABLED` | Armazena conversas, contatos e mensagens | `true` |
| `WEBHOOK_URL` | Webhook global para todas as linhas | — |
| `WEBHOOK_FORMAT` | `evolution-api` para o formato de eventos v2 | — |
| `SERVER_URL` | URL pública, usada nos eventos v2 | — |
| `METRICS_ENABLED` | Ativa ou desativa `/metrics` | `true` |

A lista completa, com RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher e novas tentativas de webhooks, está em [.env.example](./.env.example).

## Documentação da API

O Swagger UI é servido em `/swagger/index.html`. O gerenciador usa a mesma API, então tudo o que você vê nele pode ser automatizado.

## Estrutura do projeto

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Privacidade

O Enlace não envia telemetria nem dados de uso para lugar nenhum. Ele se comunica apenas com o WhatsApp e com os destinos que você configurar.

## Mantendo a sincronia com o Evolution Go

As novas funcionalidades ficam em pacotes próprios e só tocam os arquivos do upstream por meio de ganchos curtos. [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) explica como incorporar as novas versões do Evolution Go.

## Como contribuir

Issues e pull requests são bem-vindos. Execute `go test ./...` e `make manager-build` antes de abrir um pull request e mantenha completas as traduções do gerenciador nos dez idiomas.

## Créditos e licença

O Enlace é baseado no [Evolution Go](https://github.com/evolution-foundation/evolution-go) da Evolution Foundation e usa o [whatsmeow](https://github.com/tulir/whatsmeow) de Tulir Asokan.

Licenciado sob Apache 2.0 com as condições adicionais em [LICENSE](./LICENSE). "Evolution Go" é uma marca comercial da Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)); este projeto é independente e não é afiliado nem endossado pela Evolution Foundation. Os avisos de terceiros estão em [NOTICE](./NOTICE).
