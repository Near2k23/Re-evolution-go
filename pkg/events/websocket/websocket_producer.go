package websocket_producer

import (
	"net/http"
	"strings"
	"sync"
	"time"

	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/gomessguii/logger"
	"github.com/gorilla/websocket"
)

const writeTimeout = 10 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// wsClient envuelve una conexión con su propio mutex de escritura: gorilla no
// admite escrituras concurrentes en la misma conexión y Produce se llama desde
// varias goroutines a la vez.
type wsClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func (c *wsClient) writeJSON(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return c.conn.WriteJSON(v)
}

type websocketProducer struct {
	clients       map[string]map[*wsClient]struct{} // conexiones por instancia (varias permitidas)
	broadcast     map[*wsClient]struct{}            // conexiones que reciben todos los eventos
	clientsMux    sync.RWMutex
	loggerWrapper *logger_wrapper.LoggerManager
}

func NewWebsocketProducer(loggerWrapper *logger_wrapper.LoggerManager) *websocketProducer {
	return &websocketProducer{
		clients:       make(map[string]map[*wsClient]struct{}),
		broadcast:     make(map[*wsClient]struct{}),
		loggerWrapper: loggerWrapper,
	}
}

// ServeWs hace el upgrade a websocket. Con instanceId vacío la conexión recibe
// los eventos de todas las instancias; si no, solo los de esa instancia.
func ServeWs(w http.ResponseWriter, r *http.Request, instanceId string, producer *websocketProducer) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.LogError("Erro ao fazer upgrade da conexão websocket: %v", err)
		return
	}

	client := &wsClient{conn: conn}
	producer.add(instanceId, client)

	// Lee hasta que el cliente cierre para liberar la conexión.
	go func() {
		defer func() {
			producer.remove(instanceId, client)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (p *websocketProducer) add(instanceID string, client *wsClient) {
	p.clientsMux.Lock()
	defer p.clientsMux.Unlock()

	if instanceID == "" {
		p.broadcast[client] = struct{}{}
		logger.LogInfo("Cliente broadcast websocket adicionado")
		return
	}

	if p.clients[instanceID] == nil {
		p.clients[instanceID] = make(map[*wsClient]struct{})
	}
	p.clients[instanceID][client] = struct{}{}
	p.loggerWrapper.GetLogger(instanceID).LogInfo("Cliente websocket adicionado para instância: %s (%d conexões)", instanceID, len(p.clients[instanceID]))
}

func (p *websocketProducer) remove(instanceID string, client *wsClient) {
	p.clientsMux.Lock()
	defer p.clientsMux.Unlock()

	if instanceID == "" {
		delete(p.broadcast, client)
		logger.LogInfo("Cliente broadcast websocket removido")
		return
	}

	if set, ok := p.clients[instanceID]; ok {
		delete(set, client)
		if len(set) == 0 {
			delete(p.clients, instanceID)
		}
	}
	p.loggerWrapper.GetLogger(instanceID).LogInfo("Cliente websocket removido para instância: %s", instanceID)
}

// snapshot copia los destinatarios para no escribir con el lock tomado.
func (p *websocketProducer) snapshot(instanceID string) []*wsClient {
	p.clientsMux.RLock()
	defer p.clientsMux.RUnlock()

	targets := make([]*wsClient, 0, len(p.clients[instanceID])+len(p.broadcast))
	for c := range p.clients[instanceID] {
		targets = append(targets, c)
	}
	for c := range p.broadcast {
		targets = append(targets, c)
	}
	return targets
}

func (p *websocketProducer) Produce(queueName string, payload []byte, instanceID string, _ string) error {
	message := map[string]interface{}{
		"queue":   strings.ToLower(queueName),
		"payload": string(payload),
	}

	for _, client := range p.snapshot(instanceID) {
		if err := client.writeJSON(message); err != nil {
			p.loggerWrapper.GetLogger(instanceID).LogError("Erro ao enviar mensagem websocket para %s: %v", instanceID, err)
			// Una conexión rota se cierra; su goroutine de lectura la quita del mapa.
			client.conn.Close()
		}
	}

	return nil
}

// CreateGlobalQueues não faz nada para websocket producer
func (p *websocketProducer) CreateGlobalQueues() error {
	return nil
}
