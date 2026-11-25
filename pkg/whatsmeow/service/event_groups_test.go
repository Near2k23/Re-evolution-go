package whatsmeow_service

import (
	"testing"

	event_types "github.com/evolution-foundation/evolution-go/pkg/internal/event_types"
)

func TestEventGroupFor(t *testing.T) {
	cases := map[string]string{
		"Message":              event_types.MESSAGE,
		"Receipt":              event_types.READ_RECEIPT,
		"Disconnected":         event_types.CONNECTION,
		"OfflineSyncCompleted": event_types.CONNECTION,
		"PasskeyRequest":       event_types.QRCODE,
		"PasskeyConfirmation":  event_types.QRCODE,
		"PasskeyError":         event_types.QRCODE,
		"ButtonClick":          event_types.BUTTON_CLICK,
		"Picture":              event_types.PICTURE,
		"UserAbout":            event_types.USER_ABOUT,
		"Desconocido":          "",
	}
	for event, want := range cases {
		if got := eventGroupFor(event); got != want {
			t.Errorf("eventGroupFor(%q) = %q, want %q", event, got, want)
		}
	}
}

func TestEventGroupsAreValidTypes(t *testing.T) {
	for event, group := range eventGroups {
		if !event_types.IsEventType(group) {
			t.Errorf("evento %q apunta a un grupo inexistente %q", event, group)
		}
	}
}

func TestChatFromEventData(t *testing.T) {
	msg := map[string]interface{}{"data": map[string]interface{}{"Info": map[string]interface{}{"Chat": "123@g.us"}}}
	if got := chatFromEventData(msg, "Message"); got != "123@g.us" {
		t.Errorf("Message: got %q", got)
	}
	receipt := map[string]interface{}{"data": map[string]interface{}{"Chat": "1@newsletter"}}
	if got := chatFromEventData(receipt, "Receipt"); got != "1@newsletter" {
		t.Errorf("Receipt: got %q", got)
	}
	if got := chatFromEventData(map[string]interface{}{}, "Message"); got != "" {
		t.Errorf("sin data: got %q", got)
	}
}
