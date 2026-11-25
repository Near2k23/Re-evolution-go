package chatstore

import (
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Tipos de mensaje que entiende el manager.
const (
	TypeText     = "text"
	TypeImage    = "image"
	TypeVideo    = "video"
	TypeAudio    = "audio"
	TypeDocument = "document"
	TypeSticker  = "sticker"
	TypeLocation = "location"
	TypeContact  = "contact"
	TypePoll     = "poll"
	TypeOther    = "other"
)

// Content es lo que se guarda de un mensaje.
type Content struct {
	Type     string
	Text     string
	Mime     string
	FileName string
	QuotedID string
}

// Extract saca tipo, texto y datos de multimedia de un mensaje. ok=false para
// mensajes que no van al historial (reacciones, protocolo, vacíos).
func Extract(m *waE2E.Message) (c Content, ok bool) {
	if m == nil {
		return c, false
	}
	if m.GetReactionMessage() != nil || m.GetProtocolMessage() != nil || !hasContent(m) {
		return c, false
	}
	switch {
	case m.GetConversation() != "":
		c = Content{Type: TypeText, Text: m.GetConversation()}
	case m.GetExtendedTextMessage() != nil:
		x := m.GetExtendedTextMessage()
		c = Content{Type: TypeText, Text: x.GetText(), QuotedID: x.GetContextInfo().GetStanzaID()}
	case m.GetImageMessage() != nil:
		x := m.GetImageMessage()
		c = Content{Type: TypeImage, Text: x.GetCaption(), Mime: x.GetMimetype(), QuotedID: x.GetContextInfo().GetStanzaID()}
	case m.GetVideoMessage() != nil:
		x := m.GetVideoMessage()
		c = Content{Type: TypeVideo, Text: x.GetCaption(), Mime: x.GetMimetype(), QuotedID: x.GetContextInfo().GetStanzaID()}
	case m.GetPtvMessage() != nil:
		x := m.GetPtvMessage()
		c = Content{Type: TypeVideo, Mime: x.GetMimetype(), QuotedID: x.GetContextInfo().GetStanzaID()}
	case m.GetAudioMessage() != nil:
		x := m.GetAudioMessage()
		c = Content{Type: TypeAudio, Mime: x.GetMimetype(), QuotedID: x.GetContextInfo().GetStanzaID()}
	case m.GetDocumentMessage() != nil:
		x := m.GetDocumentMessage()
		name := x.GetFileName()
		if name == "" {
			name = x.GetTitle()
		}
		c = Content{Type: TypeDocument, Text: x.GetCaption(), Mime: x.GetMimetype(), FileName: name, QuotedID: x.GetContextInfo().GetStanzaID()}
	case m.GetStickerMessage() != nil:
		c = Content{Type: TypeSticker, Mime: m.GetStickerMessage().GetMimetype()}
	case m.GetLocationMessage() != nil:
		x := m.GetLocationMessage()
		c = Content{Type: TypeLocation, Text: strings.TrimSpace(x.GetName() + " " + x.GetAddress())}
	case m.GetLiveLocationMessage() != nil:
		c = Content{Type: TypeLocation, Text: m.GetLiveLocationMessage().GetCaption()}
	case m.GetContactMessage() != nil:
		c = Content{Type: TypeContact, Text: m.GetContactMessage().GetDisplayName()}
	case m.GetContactsArrayMessage() != nil:
		c = Content{Type: TypeContact, Text: m.GetContactsArrayMessage().GetDisplayName()}
	case m.GetPollCreationMessage() != nil:
		c = Content{Type: TypePoll, Text: m.GetPollCreationMessage().GetName()}
	case m.GetPollCreationMessageV2() != nil:
		c = Content{Type: TypePoll, Text: m.GetPollCreationMessageV2().GetName()}
	case m.GetPollCreationMessageV3() != nil:
		c = Content{Type: TypePoll, Text: m.GetPollCreationMessageV3().GetName()}
	case m.GetButtonsResponseMessage() != nil:
		c = Content{Type: TypeText, Text: m.GetButtonsResponseMessage().GetSelectedDisplayText()}
	case m.GetListResponseMessage() != nil:
		c = Content{Type: TypeText, Text: m.GetListResponseMessage().GetTitle()}
	case m.GetTemplateButtonReplyMessage() != nil:
		c = Content{Type: TypeText, Text: m.GetTemplateButtonReplyMessage().GetSelectedDisplayText()}
	case m.GetPollUpdateMessage() != nil, m.GetKeepInChatMessage() != nil, m.GetPinInChatMessage() != nil:
		return c, false
	default:
		c = Content{Type: TypeOther}
	}
	return c, true
}

// hasContent indica si el mensaje trae algo más que la distribución de claves
// y el contexto, que llegan solos en algunos mensajes de grupo.
func hasContent(m *waE2E.Message) bool {
	found := false
	m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		switch fd.Name() {
		case "senderKeyDistributionMessage", "messageContextInfo":
			return true
		}
		found = true
		return false
	})
	return found
}

// Preview es el texto corto que se ve en la lista de chats.
func Preview(c Content) string {
	t := strings.TrimSpace(c.Text)
	if c.Type == TypeDocument && t == "" {
		t = c.FileName
	}
	if len([]rune(t)) > 140 {
		t = string([]rune(t)[:140]) + "…"
	}
	return t
}

// ChatJID decide en qué chat se guarda un mensaje. En chats de uno a uno se
// prefiere el número de teléfono al LID cuando se conoce, para no partir la
// misma conversación en dos.
func ChatJID(src types.MessageSource) types.JID {
	chat := src.Chat.ToNonAD()
	if src.IsGroup || chat.Server != types.HiddenUserServer {
		return chat
	}
	if !src.IsFromMe && src.SenderAlt.Server == types.DefaultUserServer {
		return src.SenderAlt.ToNonAD()
	}
	if src.IsFromMe && src.RecipientAlt.Server == types.DefaultUserServer {
		return src.RecipientAlt.ToNonAD()
	}
	return chat
}

// Skip indica chats que no se guardan (estados).
func Skip(chat types.JID) bool {
	return chat == types.StatusBroadcastJID || chat.Server == types.BroadcastServer && chat.User == "status"
}
