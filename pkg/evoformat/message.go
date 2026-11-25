// Package evoformat convierte mensajes y eventos de Evolution GO al formato de
// Evolution API v2 (TypeScript), para que n8n, Typebot y demás integraciones
// hechas para v2 funcionen sin cambios. No depende de otros paquetes del
// proyecto: trabaja sobre el JSON.
package evoformat

import (
	"encoding/json"
	"strings"
	"time"
)

// M es un objeto JSON genérico.
type M = map[string]interface{}

// Get recorre claves anidadas ("Info.Chat") y devuelve nil si falta alguna.
func Get(m M, path string) interface{} {
	var cur interface{} = m
	for _, k := range strings.Split(path, ".") {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

// Str devuelve el texto en path o "".
func Str(m M, path string) string {
	switch v := Get(m, path).(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	}
	return ""
}

// Bool devuelve el booleano en path.
func Bool(m M, path string) bool {
	b, _ := Get(m, path).(bool)
	return b
}

// Obj devuelve el objeto en path o nil.
func Obj(m M, path string) M {
	o, _ := Get(m, path).(map[string]interface{})
	return o
}

// Unix convierte un tiempo RFC 3339 (como lo serializa Go) a segundos.
func Unix(v interface{}) int64 {
	switch t := v.(type) {
	case string:
		if p, err := time.Parse(time.RFC3339Nano, t); err == nil && !p.IsZero() && p.Year() > 1970 {
			return p.Unix()
		}
	case float64:
		return int64(t)
	}
	return 0
}

// MessageType es la primera clave con contenido del mensaje, como en Baileys
// (conversation, extendedTextMessage, imageMessage…).
func MessageType(msg M) string {
	best := ""
	for k, v := range msg {
		if k == "messageContextInfo" || v == nil {
			continue
		}
		if k == "conversation" {
			return k
		}
		if best == "" || k < best { // orden estable si hubiera varias
			best = k
		}
	}
	if best == "" {
		return "unknown"
	}
	return best
}

// Source adivina el dispositivo de origen por el formato del ID, como Baileys.
func Source(id string) string {
	switch {
	case len(id) > 21:
		return "android"
	case strings.HasPrefix(id, "3A"):
		return "ios"
	case strings.HasPrefix(id, "3E"):
		return "web"
	case len(id) == 0:
		return "unknown"
	}
	return "unknown"
}

// Record arma un mensaje v2 a partir del Info y Message de Go.
//
//	{key:{remoteJid, fromMe, id, participant?}, pushName, status, message,
//	 messageType, messageTimestamp, instanceId, source}
func Record(info, msg M, instanceID, status string) M {
	chat := Str(info, "Chat")
	key := M{"remoteJid": chat, "fromMe": Bool(info, "IsFromMe"), "id": Str(info, "ID")}
	if Bool(info, "IsGroup") || strings.HasSuffix(chat, "@g.us") {
		if p := Str(info, "Sender"); p != "" {
			key["participant"] = p
		}
	}
	if alt := Str(info, "SenderAlt"); alt != "" {
		key["senderPn"] = alt
	}
	if msg == nil {
		msg = M{}
	}
	return M{
		"key":              key,
		"pushName":         Str(info, "PushName"),
		"status":           status,
		"message":          msg,
		"messageType":      MessageType(msg),
		"messageTimestamp": Unix(Get(info, "Timestamp")),
		"instanceId":       instanceID,
		"source":           Source(Str(info, "ID")),
	}
}

// FromSend convierte la respuesta de /send/* ({Info, Message, …}) en un mensaje v2.
func FromSend(data M, instanceID string) M {
	return Record(Obj(data, "Info"), Obj(data, "Message"), instanceID, "PENDING")
}

// ReceiptStatus traduce el tipo de acuse de Go al estado de v2.
func ReceiptStatus(state, typ string) string {
	switch strings.ToLower(typ) {
	case "played", "played-self":
		return "PLAYED"
	case "read", "read-self":
		return "READ"
	}
	switch strings.ToLower(state) {
	case "read", "readself":
		return "READ"
	}
	return "DELIVERY_ACK"
}
