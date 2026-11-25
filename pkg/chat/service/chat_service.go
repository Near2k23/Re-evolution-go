package chat_service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatstore"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/utils"
	whatsmeow_service "github.com/evolution-foundation/evolution-go/pkg/whatsmeow/service"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type ChatService interface {
	ChatPin(data *BodyStruct, instance *instance_model.Instance) (string, error)
	ChatUnpin(data *BodyStruct, instance *instance_model.Instance) (string, error)
	ChatArchive(data *BodyStruct, instance *instance_model.Instance) (string, error)
	ChatUnarchive(data *BodyStruct, instance *instance_model.Instance) (string, error)
	ChatMute(data *BodyStruct, instance *instance_model.Instance) (string, error)
	ChatUnmute(data *BodyStruct, instance *instance_model.Instance) (string, error)
	ChatMarkUnread(data *BodyStruct, instance *instance_model.Instance) (string, error)
	HistorySyncRequest(data *HistorySyncRequestStruct, instance *instance_model.Instance) (*whatsmeow.SendResponse, error)
}

type chatService struct {
	clientPointer    map[string]*whatsmeow.Client
	whatsmeowService whatsmeow_service.WhatsmeowService
	loggerWrapper    *logger_wrapper.LoggerManager
}

type BodyStruct struct {
	Chat   string `json:"chat"`
	Number string `json:"number"` // alternativa a chat
	// Duration: segundos de silencio en /chat/mute (0 = 8 h, -1 = siempre).
	Duration int64 `json:"duration"`
	// Read: en /chat/markunread, true marca como leído.
	Read bool `json:"read"`
}

type HistorySyncRequestStruct struct {
	MessageInfo *types.MessageInfo `json:"messageInfo"`
	Count       int                `json:"count"`
}

func (c *chatService) ensureClientConnected(instanceId string) (*whatsmeow.Client, error) {
	client := c.clientPointer[instanceId]
	c.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking client connection status - Client exists: %v", instanceId, client != nil)

	if client == nil {
		c.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] No client found, attempting to start new instance", instanceId)
		err := c.whatsmeowService.StartInstance(instanceId)
		if err != nil {
			c.loggerWrapper.GetLogger(instanceId).LogError("[%s] Failed to start instance: %v", instanceId, err)
			return nil, errors.New("no active session found")
		}

		c.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Instance started, waiting 2 seconds...", instanceId)
		time.Sleep(2 * time.Second)

		client = c.clientPointer[instanceId]
		c.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking new client - Exists: %v, Connected: %v",
			instanceId,
			client != nil,
			client != nil && client.IsConnected())

		if client == nil || !client.IsConnected() {
			c.loggerWrapper.GetLogger(instanceId).LogError("[%s] New client validation failed - Exists: %v, Connected: %v",
				instanceId,
				client != nil,
				client != nil && client.IsConnected())
			return nil, errors.New("no active session found")
		}
	} else if !client.IsConnected() {
		c.loggerWrapper.GetLogger(instanceId).LogError("[%s] Existing client is disconnected - Connected status: %v",
			instanceId,
			client.IsConnected())
		return nil, errors.New("client disconnected")
	}

	c.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Client successfully validated - Connected: %v", instanceId, client.IsConnected())
	return client, nil
}

// target resuelve el chat del cuerpo (chat o number; número o JID).
func target(data *BodyStruct) (types.JID, error) {
	raw := data.Chat
	if raw == "" {
		raw = data.Number
	}
	if raw == "" {
		return types.JID{}, errors.New("chat is required")
	}
	if !strings.Contains(raw, "@") {
		jid, err := utils.CreateJID(raw)
		if err != nil {
			return types.JID{}, err
		}
		raw = jid
	}
	jid, ok := utils.ParseJID(raw)
	if !ok {
		return types.JID{}, errors.New("invalid chat")
	}
	jid = jid.ToNonAD()
	jid.User = strings.TrimPrefix(jid.User, "+") // CreateJID antepone «+»
	return jid, nil
}

// lastKey es el último mensaje del chat según el historial (WhatsApp lo usa
// para archivar y marcar como leído/no leído).
func lastKey(instanceID string, chat types.JID) (time.Time, *waCommon.MessageKey) {
	m, ok := chatstore.LastOf(instanceID, chat.String())
	if !ok {
		return time.Now(), nil
	}
	key := &waCommon.MessageKey{RemoteJID: proto.String(chat.String()), FromMe: proto.Bool(m.FromMe), ID: proto.String(m.ID)}
	if chat.Server == types.GroupServer && !m.FromMe && m.Sender != "" {
		key.Participant = proto.String(m.Sender)
	}
	return m.Timestamp, key
}

// applyState envía un cambio de estado del chat. Si faltan las claves de
// sincronización, pide el estado completo y reintenta una vez.
func (c *chatService) applyState(data *BodyStruct, instance *instance_model.Instance, action string, build func(types.JID) appstate.PatchInfo) (string, error) {
	client, err := c.ensureClientConnected(instance.Id)
	if err != nil {
		return "", err
	}
	if !client.IsLoggedIn() || client.Store == nil || client.Store.ID == nil {
		return "", errors.New("instance is not logged in")
	}
	jid, err := target(data)
	if err != nil {
		return "", err
	}
	patch := build(jid)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.SendAppState(ctx, patch)
	if err != nil && strings.Contains(err.Error(), "app state key") {
		c.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] %s chat: %v; syncing app state and retrying", instance.Id, action, err)
		_ = client.FetchAppState(ctx, patch.Type, true, false)
		err = client.SendAppState(ctx, patch)
	}
	if err != nil {
		c.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error %s chat: %v", instance.Id, action, err)
		return "", err
	}
	return time.Now().Format(time.RFC3339), nil
}

func (c *chatService) ChatPin(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "pin", func(j types.JID) appstate.PatchInfo { return appstate.BuildPin(j, true) })
}

func (c *chatService) ChatUnpin(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "unpin", func(j types.JID) appstate.PatchInfo { return appstate.BuildPin(j, false) })
}

func (c *chatService) ChatArchive(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "archive", func(j types.JID) appstate.PatchInfo {
		ts, key := lastKey(instance.Id, j)
		return appstate.BuildArchive(j, true, ts, key)
	})
}

func (c *chatService) ChatUnarchive(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "unarchive", func(j types.JID) appstate.PatchInfo {
		ts, key := lastKey(instance.Id, j)
		return appstate.BuildArchive(j, false, ts, key)
	})
}

// ChatMute silencia por Duration segundos (8 h si no se indica; -1 = siempre).
func (c *chatService) ChatMute(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "mute", func(j types.JID) appstate.PatchInfo {
		if data.Duration < 0 {
			forever := int64(-1)
			return appstate.BuildMuteAbs(j, true, &forever)
		}
		d := time.Duration(data.Duration) * time.Second
		if d == 0 {
			d = 8 * time.Hour
		}
		return appstate.BuildMute(j, true, d)
	})
}

func (c *chatService) ChatUnmute(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "unmute", func(j types.JID) appstate.PatchInfo { return appstate.BuildMute(j, false, 0) })
}

// ChatMarkUnread marca el chat como no leído (o leído con Read=true) en todos los dispositivos.
func (c *chatService) ChatMarkUnread(data *BodyStruct, instance *instance_model.Instance) (string, error) {
	return c.applyState(data, instance, "mark unread", func(j types.JID) appstate.PatchInfo {
		ts, key := lastKey(instance.Id, j)
		return appstate.BuildMarkChatAsRead(j, data.Read, ts, key)
	})
}

func (c *chatService) HistorySyncRequest(data *HistorySyncRequestStruct, instance *instance_model.Instance) (*whatsmeow.SendResponse, error) {
	client, err := c.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	messageInfo := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     data.MessageInfo.Chat,
			IsFromMe: data.MessageInfo.IsFromMe,
			IsGroup:  data.MessageInfo.IsGroup,
		},
		ID:        data.MessageInfo.ID,
		Timestamp: data.MessageInfo.Timestamp,
	}

	histRequest := client.BuildHistorySyncRequest(&messageInfo, data.Count)

	res, err := client.SendMessage(context.Background(), messageInfo.Chat, histRequest, whatsmeow.SendRequestExtra{Peer: true})
	if err != nil {
		c.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error history sync request: %v", instance.Id, err)
		return nil, err
	}

	return &res, nil
}

func NewChatService(
	clientPointer map[string]*whatsmeow.Client,
	whatsmeowService whatsmeow_service.WhatsmeowService,
	loggerWrapper *logger_wrapper.LoggerManager,
) ChatService {
	return &chatService{
		clientPointer:    clientPointer,
		whatsmeowService: whatsmeowService,
		loggerWrapper:    loggerWrapper,
	}
}
