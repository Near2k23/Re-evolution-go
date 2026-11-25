package evoformat

import (
	"encoding/json"
	"strings"
	"time"
)

// Nombres de eventos de Evolution API v2.
const (
	QrcodeUpdated     = "qrcode.updated"
	ConnectionUpdate  = "connection.update"
	MessagesSet       = "messages.set"
	MessagesUpsert    = "messages.upsert"
	MessagesUpdate    = "messages.update"
	SendMessage       = "send.message"
	ContactsUpdate    = "contacts.update"
	PresenceUpdate    = "presence.update"
	ChatsUpdate       = "chats.update"
	GroupsUpsert      = "groups.upsert"
	GroupsUpdate      = "groups.update"
	Call              = "call"
	LabelsEdit        = "labels.edit"
	LabelsAssociation = "labels.association"
)

// goToV2 asigna cada evento de Evolution GO a su nombre en v2. Los que no
// están aquí no se envían a un webhook en formato v2.
var goToV2 = map[string]string{
	"Message":              MessagesUpsert,
	"SendMessage":          SendMessage,
	"Receipt":              MessagesUpdate,
	"QRCode":               QrcodeUpdated,
	"Connected":            ConnectionUpdate,
	"PairSuccess":          ConnectionUpdate,
	"LoggedOut":            ConnectionUpdate,
	"Disconnected":         ConnectionUpdate,
	"ConnectFailure":       ConnectionUpdate,
	"TemporaryBan":         ConnectionUpdate,
	"Presence":             PresenceUpdate,
	"ChatPresence":         PresenceUpdate,
	"HistorySync":          MessagesSet,
	"Archive":              ChatsUpdate,
	"Contact":              ContactsUpdate,
	"PushName":             ContactsUpdate,
	"GroupInfo":            GroupsUpdate,
	"JoinedGroup":          GroupsUpsert,
	"CallOffer":            Call,
	"CallAccept":           Call,
	"CallTerminate":        Call,
	"CallOfferNotice":      Call,
	"LabelEdit":            LabelsEdit,
	"LabelAssociationChat": LabelsAssociation,
}

// V2Event devuelve el nombre v2 de un evento de Go ("" si no tiene).
func V2Event(goEvent string) string { return goToV2[goEvent] }

// Upper convierte "messages.upsert" en "MESSAGES_UPSERT".
func Upper(v2 string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(v2))
}

// Path es el sufijo de URL de un evento con «webhook por eventos» en v2
// ("messages.upsert" → "messages-upsert").
func Path(v2 string) string {
	return strings.ToLower(strings.ReplaceAll(Upper(v2), "_", "-"))
}

// v2Groups traduce los eventos de suscripción de v2 a los grupos de Go.
var v2Groups = map[string][]string{
	"MESSAGES_UPSERT":           {"MESSAGE"},
	"MESSAGES_SET":              {"HISTORY_SYNC"},
	"MESSAGES_UPDATE":           {"READ_RECEIPT"},
	"SEND_MESSAGE":              {"SEND_MESSAGE"},
	"SEND_MESSAGE_UPDATE":       {"READ_RECEIPT"},
	"QRCODE_UPDATED":            {"QRCODE"},
	"CONNECTION_UPDATE":         {"CONNECTION"},
	"PRESENCE_UPDATE":           {"PRESENCE", "CHAT_PRESENCE"},
	"CHATS_UPDATE":              {"CHAT_PRESENCE"},
	"CHATS_UPSERT":              {"CHAT_PRESENCE"},
	"CONTACTS_UPDATE":           {"CONTACT"},
	"CONTACTS_UPSERT":           {"CONTACT"},
	"CONTACTS_SET":              {"CONTACT"},
	"GROUPS_UPSERT":             {"GROUP"},
	"GROUPS_UPDATE":             {"GROUP"},
	"GROUP_PARTICIPANTS_UPDATE": {"GROUP"},
	"CALL":                      {"CALL"},
	"LABELS_EDIT":               {"LABEL"},
	"LABELS_ASSOCIATION":        {"LABEL"},
}

// GroupsFor traduce una lista de eventos v2 a grupos de suscripción de Go.
// Sin eventos o con alguno desconocido que lo pida todo, devuelve ["ALL"].
func GroupsFor(events []string) []string {
	if len(events) == 0 {
		return []string{"ALL"}
	}
	seen := map[string]bool{}
	var out []string
	for _, e := range events {
		for _, g := range v2Groups[Upper(e)] {
			if !seen[g] {
				seen[g] = true
				out = append(out, g)
			}
		}
	}
	if len(out) == 0 {
		return []string{"ALL"}
	}
	return out
}

// EventsFor es la inversa aproximada: grupos de Go → eventos v2.
func EventsFor(groups []string) []string {
	var out []string
	seen := map[string]bool{}
	all := len(groups) == 0
	has := map[string]bool{}
	for _, g := range groups {
		g = strings.TrimSpace(g)
		has[g] = true
		if g == "ALL" {
			all = true
		}
	}
	for _, name := range []string{"MESSAGES_UPSERT", "MESSAGES_SET", "MESSAGES_UPDATE", "SEND_MESSAGE", "QRCODE_UPDATED",
		"CONNECTION_UPDATE", "PRESENCE_UPDATE", "CHATS_UPDATE", "CONTACTS_UPDATE", "GROUPS_UPSERT", "GROUPS_UPDATE", "CALL",
		"LABELS_EDIT", "LABELS_ASSOCIATION"} {
		for _, g := range v2Groups[name] {
			if (all || has[g]) && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// Meta son los datos del sobre v2 que no vienen en el evento de Go.
type Meta struct {
	ServerURL   string // SERVER_URL
	Sender      string // JID de la línea
	Destination string // URL del webhook
}

// Converted es un evento ya en formato v2.
type Converted struct {
	Event   string // "messages.upsert"
	Payload []byte
}

// Convert traduce un evento de Go ({event, data, instanceId, instanceName,
// instanceToken}) a uno o varios eventos v2 ({event, instance, data,
// destination, date_time, sender, server_url, apikey}). Devuelve nil si el
// evento no tiene equivalente.
func Convert(payload []byte, meta Meta) []Converted {
	var in M
	if err := json.Unmarshal(payload, &in); err != nil {
		return nil
	}
	goEvent := Str(in, "event")
	v2 := goToV2[goEvent]
	if v2 == "" {
		return nil
	}
	instanceID := Str(in, "instanceId")
	name := Str(in, "instanceName")
	data, _ := in["data"].(map[string]interface{})
	if data == nil {
		data = M{}
	}

	var datas []interface{}
	switch goEvent {
	case "Message":
		rec := Record(Obj(data, "Info"), Obj(data, "Message"), instanceID, "DELIVERY_ACK")
		if Bool(data, "Info.IsFromMe") {
			rec["status"] = "SERVER_ACK"
		}
		datas = append(datas, rec)
	case "SendMessage":
		rec := Record(Obj(data, "Info"), Obj(data, "Message"), instanceID, "PENDING")
		datas = append(datas, rec)
	case "Receipt":
		status := ReceiptStatus(Str(in, "state"), Str(data, "Type"))
		ids, _ := data["MessageIDs"].([]interface{})
		for _, id := range ids {
			s, _ := id.(string)
			datas = append(datas, M{
				"messageId":   s,
				"keyId":       s,
				"remoteJid":   Str(data, "Chat"),
				"fromMe":      true,
				"participant": Str(data, "Sender"),
				"status":      status,
				"instanceId":  instanceID,
			})
		}
	case "QRCode":
		datas = append(datas, M{"qrcode": M{"instance": name, "pairingCode": nil, "code": Str(data, "code"), "base64": Str(data, "qrcode")}})
	case "Connected", "PairSuccess":
		datas = append(datas, M{"instance": name, "state": "open", "statusReason": 200, "wuid": meta.Sender})
	case "LoggedOut":
		datas = append(datas, M{"instance": name, "state": "close", "statusReason": 401})
	case "TemporaryBan":
		datas = append(datas, M{"instance": name, "state": "close", "statusReason": 403})
	case "Disconnected", "ConnectFailure":
		datas = append(datas, M{"instance": name, "state": "connecting", "statusReason": 428})
	case "Presence":
		jid := Str(data, "From")
		p := "available"
		if Str(in, "state") == "offline" || Bool(data, "Unavailable") {
			p = "unavailable"
		}
		datas = append(datas, M{"id": jid, "presences": M{jid: M{"lastKnownPresence": p}}})
	case "ChatPresence":
		chat := Str(data, "Chat")
		who := Str(data, "Sender")
		if who == "" {
			who = chat
		}
		p := "paused"
		if Str(data, "State") == "composing" {
			p = "composing"
			if Str(data, "Media") == "audio" {
				p = "recording"
			}
		}
		datas = append(datas, M{"id": chat, "presences": M{who: M{"lastKnownPresence": p}}})
	case "Contact", "PushName":
		jid := Str(data, "JID")
		push := Str(data, "NewPushName")
		if push == "" {
			push = Str(data, "Action.firstName")
		}
		datas = append(datas, []interface{}{M{"remoteJid": jid, "pushName": push, "instanceId": instanceID}})
	case "CallOffer", "CallAccept", "CallTerminate", "CallOfferNotice":
		status := map[string]string{"CallOffer": "offer", "CallAccept": "accept", "CallTerminate": "terminate", "CallOfferNotice": "offer"}[goEvent]
		datas = append(datas, M{"id": Str(data, "CallID"), "from": Str(data, "CallCreator"), "status": status,
			"date": Unix(Get(data, "Timestamp")), "isGroup": Str(data, "GroupJID") != "", "isVideo": strings.Contains(string(payload), `"Tag":"video"`)})
	default:
		datas = append(datas, data)
	}

	now := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	out := make([]Converted, 0, len(datas))
	for _, d := range datas {
		env := M{
			"event":       v2,
			"instance":    name,
			"data":        d,
			"destination": meta.Destination,
			"date_time":   now,
			"sender":      meta.Sender,
			"server_url":  meta.ServerURL,
			"apikey":      Str(in, "instanceToken"),
		}
		b, err := json.Marshal(env)
		if err == nil {
			out = append(out, Converted{Event: v2, Payload: b})
		}
	}
	return out
}
