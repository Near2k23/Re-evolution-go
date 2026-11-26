<h1 align="center">Enlace</h1>

<p align="center">
  <b>Une API WhatsApp auto-hébergée avec un gestionnaire moderne, l’historique des conversations, des bots et la compatibilité Evolution API v2.</b><br/>
  Open source · Écrit en Go · Basé sur Evolution Go
</p>

<p align="center">
  <a href="./README.md">English</a> ·
  <a href="./README.es.md">Español</a> ·
  <a href="./README.pt.md">Português</a> ·
  <b>Français</b> ·
  <a href="./README.it.md">Italiano</a> ·
  <a href="./README.de.md">Deutsch</a> ·
  <a href="./README.ru.md">Русский</a> ·
  <a href="./README.zh.md">中文</a> ·
  <a href="./README.ja.md">日本語</a> ·
  <a href="./README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./docs/images/dashboard.png" alt="Tableau de bord d’Enlace : lignes opérationnelles, coupures, livraison des webhooks et disponibilité heure par heure" width="100%" />
</p>

---

## Pourquoi Enlace

Gérer de nombreux numéros WhatsApp consiste surtout à savoir **quelles lignes sont en bonne santé, lesquelles ont décroché puis sont revenues, et lesquelles ont besoin d’une intervention humaine**. Enlace est construit autour de cela :

- **Voyez toutes les lignes d’un coup d’œil.** Les lignes sont réparties en *connectées*, *reconnectées* (tombées puis revenues d’elles-mêmes), *en reconnexion* et *hors service*, avec une carte de disponibilité heure par heure et un journal de chaque coupure.
- **Réparez-les depuis un seul endroit.** Reconnectez, associez par QR ou code d’appairage, déconnectez, définissez un proxy ou modifiez les paramètres sans toucher à l’API.
- **Lisez et répondez aux conversations.** Chaque ligne conserve ses conversations, contacts et messages dans PostgreSQL, et le gestionnaire les affiche comme une application de messagerie.
- **Automatisez les réponses.** Des bots par ligne avec n8n, OpenAI, Typebot, Dify, Flowise ou votre propre endpoint HTTP, ainsi que Chatwoot dans les deux sens.
- **Branchez ce que vous utilisez déjà.** Webhooks signés avec nouvelles tentatives, RabbitMQ, NATS, WebSocket, Amazon SQS, Kafka, Pusher et métriques Prometheus.
- **Conservez vos intégrations Evolution API v2.** Les routes et les noms d’événements v2 fonctionnent sur le même serveur : n8n, Typebot et vos scripts existants tournent sans modification.
- **Dans votre langue.** Le gestionnaire parle anglais, espagnol, portugais, français, italien, allemand, russe, chinois, japonais et coréen, sur ordinateur comme sur mobile.

## Captures d’écran

| | |
|---|---|
| <img src="./docs/images/lines.png" alt="Lignes regroupées par état" /> | <img src="./docs/images/line-detail.png" alt="Détail d’une ligne avec coupures et carte de connexion sur 7 jours" /> |
| **Lignes** regroupées par état, filtrables par vos propres groupes | **Détail d’une ligne** : coupures, coupure la plus longue et les 7 derniers jours heure par heure |
| <img src="./docs/images/chats.png" alt="Conversations avec historique" /> | <img src="./docs/images/chat-menu.png" alt="Actions de conversation" /> |
| **Conversations** stockées par ligne, avec accusés de réception et de lecture | **Actions de conversation** : archiver, épingler, mettre en sourdine, marquer comme non lu, messages éphémères |
| <img src="./docs/images/bots.png" alt="Bots par ligne" /> | <img src="./docs/images/webhook.png" alt="Paramètres du webhook" /> |
| **Bots** par ligne, avec sessions et Chatwoot | **Webhooks** : événements, format v2, fichiers en base64, signature et en-têtes |

<p align="center">
  <img src="./docs/images/mobile-dashboard.png" alt="Tableau de bord sur mobile" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-chat.png" alt="Conversation sur mobile" width="30%" />
  &nbsp;
  <img src="./docs/images/mobile-lines-ja.png" alt="Lignes en japonais sur mobile" width="30%" />
</p>
<p align="center"><sub>Le même gestionnaire sur un téléphone, ici en anglais et en japonais.</sub></p>

### Principes de conception

- **Discret par défaut, visible quand ça compte.** Les lignes en bonne santé s’effacent à l’arrière-plan ; les coupures et les lignes qui ont besoin d’une intervention humaine passent en premier, avec une action claire.
- **Aucune décoration.** Ni bordures ni séparateurs, ni dégradés : des surfaces douces, des espacements généreux et de la couleur uniquement pour l’état (vert connectée, bleu reconnectée, orange en reconnexion, rouge hors service).
- **Des chiffres lisibles.** Les numéros de téléphone, les tokens et les heures utilisent une police à chasse fixe ; les compteurs sont grands et sobres.
- **La même application sur chaque écran.** Une barre latérale sur ordinateur et une barre d’onglets en bas sur mobile, avec les mêmes pages et les mêmes actions.

---

## Démarrage rapide

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

Créez les deux bases de données une seule fois (`evogo_auth` et `evogo_users`), lancez le service avec `docker compose up -d` et ouvrez **http://localhost:8080/manager**. Connectez-vous avec l’URL du serveur et votre `GLOBAL_API_KEY`, créez une ligne et associez-la avec le code QR.

### Développement local

```bash
make setup
cp .env.example .env      # set GLOBAL_API_KEY and the two Postgres URLs
make manager-install && make manager-build
make dev
```

`make manager-dev` lance le gestionnaire avec Vite sur le port 5173. Exécutez `make help` pour voir toutes les commandes.

---

## Fonctionnalités

### Lignes et santé de la connexion

- `GET /instance/overview` renvoie toutes les lignes avec leur état, les coupures des dernières 24 h, la coupure la plus longue et une carte heure par heure. `GET /instance/history/:id` renvoie le journal des coupures.
- `POST /instance/reconnect/:id` tente de rétablir une ligne tombée sans la dissocier.
- Les lignes démarrent avec le serveur (`CONNECT_ON_STARTUP`, activé par défaut) et peuvent être organisées dans vos propres **groupes** (`/line-groups`).

### Webhooks

Chaque ligne possède son URL et ses événements (définis avec `/instance/connect`), ainsi que ses propres paramètres dans `GET/PUT /webhook/:instanceId` :

- **En-têtes supplémentaires**, **une URL par événement** (`…/message`, `…/send-message`) et **fichiers en base64** (`"base64": true`).
- **Signature HMAC** : `X-Webhook-Signature: sha256=hex(HMAC_SHA256(secret, timestamp + "." + body))` avec `X-Webhook-Timestamp`. Chaque requête transporte aussi `X-Webhook-Event` et un `X-Webhook-Delivery` unique.
- **Nouvelles tentatives** avec des délais croissants (1 s, 2 s, 4 s… jusqu’à 60 s, 8 tentatives ; les réponses 4xx ne sont pas réessayées, sauf 408/425/429), un journal des livraisons (`/webhook/:id/deliveries`) et un événement de test (`POST /webhook/:id/test`).
- **Format Evolution API v2** par ligne ou pour le webhook global (`WEBHOOK_FORMAT=evolution-api`).

### Historique des conversations

Avec `CHAT_STORE_ENABLED=true` (par défaut), chaque ligne stocke ses conversations, contacts et messages : ce qu’elle reçoit, ce qu’elle envoie via l’API et l’historique que WhatsApp transmet lorsque la ligne est associée.

| Méthode | Route | Usage |
|---|---|---|
| GET | `/store/:id/chats?q=&archived=` | Conversations, épinglées en premier |
| GET | `/store/:id/messages?chat=<jid>&before=` | Messages, les plus récents en premier |
| GET | `/store/:id/contacts?q=` | Contacts connus |
| POST | `/store/:id/read` | Marquer comme lu et envoyer les accusés de lecture |

Les messages sont conservés pendant `CHAT_STORE_RETENTION_DAYS` jours (90 par défaut ; 0 = indéfiniment).

### Actions de conversation, groupes et contacts

- `POST /chat/archive|unarchive|pin|unpin|mute|unmute|markunread` avec `{"chat": "<jid>"}` ou `{"number": "…"}`. `mute` accepte `duration` en secondes (8 h par défaut, `-1` = toujours).
- `POST /group/inviteinfo`, `POST /group/ephemeral` (messages éphémères), `POST /user/business` (profil professionnel).
- `POST /send/contact` accepte `vcards: [...]` pour envoyer plusieurs contacts dans un seul message.

### Bots

Plusieurs bots par ligne (`/bots/:instanceId`). Dans chaque conversation, un seul bot parle à la fois : le premier, par ordre de priorité, dont le déclencheur correspond (n’importe quel message, ou un mot-clé avec `contains`, `equals`, `startsWith`, `endsWith` ou `regex`).

| Type | Ce qu’il fait |
|---|---|
| `http` | Envoie chaque message en JSON à votre URL et répond avec ce qu’elle renvoie |
| `n8n` | Idem, vers un nœud Webhook n8n |
| `openai` | Chat Completions avec l’historique récent ; transcrit les messages vocaux |
| `typebot` | Exécute un flux Typebot publié |
| `dify` | Applications de chat, d’agent, de workflow ou de génération de texte |
| `flowise` | Un chatflow Flowise |

Options communes : répondre dans les groupes, se mettre en pause quand une personne répond, délai de saisie, regroupement des messages rapprochés, découpage des longues réponses, mot-clé de fin, expiration et conversations ignorées.

### Chatwoot

`PUT /chatwoot/:instanceId` avec l’URL, le compte et le token de votre Chatwoot. Enlace crée une boîte de réception de type API, reproduit chaque conversation WhatsApp (avec ses pièces jointes) sous forme de conversation et envoie les réponses des agents via la ligne.

### Compatibilité Evolution API v2

Les routes v2 cohabitent avec celles de Go. Les lignes sont désignées par leur **nom** et `apikey` accepte la clé globale ou le token de la ligne.

| Groupe | Routes |
|---|---|
| Instance | `create`, `fetchInstances`, `connect`, `connectionState`, `restart`, `logout`, `delete`, `setPresence` |
| Messages | `sendText`, `sendMedia`, `sendPtv`, `sendWhatsAppAudio`, `sendSticker`, `sendLocation`, `sendContact`, `sendReaction`, `sendPoll`, `sendList`, `sendButtons`, `sendStatus` |
| Conversation | `whatsappNumbers`, `markMessageAsRead`, `archiveChat`, `markChatUnread`, `findChats`, `findMessages`, `findContacts`, `getBase64FromMediaMessage`, `fetchProfile`, … |
| Groupes | `create`, `updateGroupSubject`, `updateParticipant`, `fetchAllGroups`, `inviteCode`, `inviteInfo`, `toggleEphemeral`, `leaveGroup`, … |
| Paramètres | `settings`, `proxy`, `webhook`, `websocket`, `rabbitmq`, `nats` (`set` et `find`) |
| Bots | `typebot`, `openai` (avec `creds`), `dify`, `flowise`, `n8n`, `evolutionBot` et `chatwoot` |

Les événements peuvent être envoyés avec les noms et l’enveloppe de la v2 (`messages.upsert`, `connection.update`, …). Les différences avec la v2 sont listées dans [docs/COMPAT.md](./docs/COMPAT.md).

### Destinations des événements

En plus des webhooks, de RabbitMQ, de NATS et du WebSocket, les événements de chaque ligne peuvent être envoyés vers :

| Destination | Variables principales |
|---|---|
| Amazon SQS | `SQS_ENABLED`, `SQS_QUEUE_URL`, `SQS_REGION`, `SQS_ACCESS_KEY_ID`, `SQS_SECRET_ACCESS_KEY` |
| Kafka | `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_TOPIC_PER_EVENT`, `KAFKA_SASL_*`, `KAFKA_TLS` |
| Pusher / Soketi | `PUSHER_ENABLED`, `PUSHER_APP_ID`, `PUSHER_KEY`, `PUSHER_SECRET`, `PUSHER_CLUSTER` ou `PUSHER_HOST` |

Chacune accepte `<DESTINATION>_EVENTS` (filtre) et `<DESTINATION>_FORMAT=evolution-api`.

### Métriques

`GET /metrics` expose des métriques Prometheus : lignes par état, messages reçus et envoyés, appels, livraisons de webhooks et temps de livraison, ainsi que des données sur le processus. L’accès requiert la clé globale (`apikey` ou `Authorization: Bearer`) ou une authentification basique avec `METRICS_USER` / `METRICS_PASSWORD`.

---

## Configuration

| Variable | Description | Valeur par défaut |
|---|---|---|
| `GLOBAL_API_KEY` | Clé d’administration pour l’API et le gestionnaire | **obligatoire** |
| `POSTGRES_AUTH_DB`, `POSTGRES_USERS_DB` | URL de connexion PostgreSQL | **obligatoire** |
| `SERVER_PORT` | Port HTTP | `8080` |
| `CLIENT_NAME` | Regroupe les lignes de ce serveur (utile lorsque plusieurs serveurs partagent une base de données) | — |
| `CONNECT_ON_STARTUP` | Reconnecte toutes les lignes au démarrage | `true` |
| `DATABASE_SAVE_MESSAGES` | Enregistre l’état de livraison des messages (`true` ou `false`) | **obligatoire** |
| `CHAT_STORE_ENABLED` | Enregistre les conversations, contacts et messages | `true` |
| `WEBHOOK_URL` | Webhook global pour toutes les lignes | — |
| `WEBHOOK_FORMAT` | `evolution-api` pour le format d’événements v2 | — |
| `SERVER_URL` | URL publique, utilisée dans les événements v2 | — |
| `METRICS_ENABLED` | Active ou désactive `/metrics` | `true` |

La liste complète, avec RabbitMQ, NATS, MinIO/S3, SQS, Kafka, Pusher et les nouvelles tentatives des webhooks, se trouve dans [.env.example](./.env.example).

## Documentation de l’API

Swagger UI est servi sur `/swagger/index.html`. Le gestionnaire utilise la même API : tout ce que vous y voyez peut donc être automatisé.

## Structure du projet

```
cmd/evolution-go/     entry point
pkg/                  API (Gin), WhatsApp (whatsmeow) and the features above:
  chatstore/ webhooks/ chatbot/ chatwoot/ compat/ evoformat/ sinks/ metrics/ …
manager/              Enlace manager (Vite + React + TypeScript)
docs/                 Swagger, screenshots and guides
```

## Confidentialité

Enlace n’envoie aucune télémétrie ni donnée d’utilisation où que ce soit. Il communique uniquement avec WhatsApp et avec les destinations que vous configurez.

## Rester synchronisé avec Evolution Go

Les nouvelles fonctionnalités résident dans leurs propres paquets et ne touchent les fichiers upstream que par de courts points d’accroche. [docs/SYNC_UPSTREAM.md](./docs/SYNC_UPSTREAM.md) explique comment intégrer les nouvelles versions d’Evolution Go.

## Contribuer

Les issues et les pull requests sont les bienvenues. Veuillez exécuter `go test ./...` et `make manager-build` avant d’ouvrir une pull request, et maintenir complètes les traductions du gestionnaire dans les dix langues.

## Crédits et licence

Enlace est basé sur [Evolution Go](https://github.com/evolution-foundation/evolution-go) d’Evolution Foundation et utilise [whatsmeow](https://github.com/tulir/whatsmeow) de Tulir Asokan.

Distribué sous licence Apache 2.0 avec les conditions supplémentaires figurant dans [LICENSE](./LICENSE). « Evolution Go » est une marque d’Evolution Foundation ([TRADEMARKS.md](./TRADEMARKS.md)) ; ce projet est indépendant et n’est ni affilié à Evolution Foundation ni approuvé par celle-ci. Les mentions relatives aux tiers figurent dans [NOTICE](./NOTICE).
