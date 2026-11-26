package whatsmeow_service

import event_types "github.com/evolution-foundation/evolution-go/pkg/internal/event_types"

// eventGroups asigna cada evento emitido a su grupo de suscripción
// (instance.Events, AMQP_GLOBAL_EVENTS, NATS_GLOBAL_EVENTS). Es la única tabla:
// un evento que no esté aquí solo llega a quien se suscribe a ALL.
var eventGroups = map[string]string{
	"Message":                 event_types.MESSAGE,
	"SendMessage":             event_types.SEND_MESSAGE,
	"Receipt":                 event_types.READ_RECEIPT,
	"Presence":                event_types.PRESENCE,
	"HistorySync":             event_types.HISTORY_SYNC,
	"ChatPresence":            event_types.CHAT_PRESENCE,
	"Archive":                 event_types.CHAT_PRESENCE,
	"CallOffer":               event_types.CALL,
	"CallAccept":              event_types.CALL,
	"CallTerminate":           event_types.CALL,
	"CallOfferNotice":         event_types.CALL,
	"CallRelayLatency":        event_types.CALL,
	"Connected":               event_types.CONNECTION,
	"PairSuccess":             event_types.CONNECTION,
	"TemporaryBan":            event_types.CONNECTION,
	"LoggedOut":               event_types.CONNECTION,
	"ConnectFailure":          event_types.CONNECTION,
	"Disconnected":            event_types.CONNECTION,
	"OfflineSyncCompleted":    event_types.CONNECTION,
	"LabelEdit":               event_types.LABEL,
	"LabelAssociationChat":    event_types.LABEL,
	"LabelAssociationMessage": event_types.LABEL,
	"Contact":                 event_types.CONTACT,
	"PushName":                event_types.CONTACT,
	"Picture":                 event_types.PICTURE,
	"UserAbout":               event_types.USER_ABOUT,
	"GroupInfo":               event_types.GROUP,
	"JoinedGroup":             event_types.GROUP,
	"NewsletterJoin":          event_types.NEWSLETTER,
	"NewsletterLeave":         event_types.NEWSLETTER,
	"QRCode":                  event_types.QRCODE,
	"QRTimeout":               event_types.QRCODE,
	"QRSuccess":               event_types.QRCODE,
	// El emparejamiento por passkey es parte del flujo de vinculación (QR/código).
	"PasskeyRequest":      event_types.QRCODE,
	"PasskeyConfirmation": event_types.QRCODE,
	"PasskeyError":        event_types.QRCODE,
	"ButtonClick":         event_types.BUTTON_CLICK,
}

// EventGroupFor es eventGroupFor para otros paquetes.
func EventGroupFor(eventType string) string { return eventGroups[eventType] }

// eventGroupFor devuelve el grupo de suscripción de un evento, o "" si no tiene.
func eventGroupFor(eventType string) string {
	return eventGroups[eventType]
}
