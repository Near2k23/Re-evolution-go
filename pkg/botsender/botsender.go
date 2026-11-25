// Package botsender adapta el servicio de envío para bots y Chatwoot: así sus
// mensajes pasan por el mismo camino que la API (webhooks, historial).
package botsender

import (
	"fmt"
	"strings"

	instance_service "github.com/evolution-foundation/evolution-go/pkg/instance/service"
	send_service "github.com/evolution-foundation/evolution-go/pkg/sendMessage/service"
)

type Sender struct {
	send      send_service.SendService
	instances instance_service.InstanceService
}

func New(send send_service.SendService, instances instance_service.InstanceService) *Sender {
	return &Sender{send: send, instances: instances}
}

func (s *Sender) SendText(instanceID, jid, text string) error {
	inst, err := s.instances.Info(instanceID)
	if err != nil || inst == nil {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	_, err = s.send.SendText(&send_service.TextStruct{Number: jid, Text: text}, inst)
	return err
}

func (s *Sender) SendMedia(instanceID, jid, url, mediaType, caption, fileName string) error {
	inst, err := s.instances.Info(instanceID)
	if err != nil || inst == nil {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	switch mediaType {
	case "image", "video", "audio", "document":
	default:
		mediaType = "document"
	}
	if fileName == "" && mediaType == "document" {
		fileName = url[strings.LastIndex(url, "/")+1:]
		if i := strings.IndexAny(fileName, "?#"); i >= 0 {
			fileName = fileName[:i]
		}
	}
	_, err = s.send.SendMediaUrl(&send_service.MediaStruct{Number: jid, Url: url, Type: mediaType, Caption: caption, Filename: fileName}, inst)
	return err
}
