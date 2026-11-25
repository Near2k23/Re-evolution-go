package webhook_producer

import (
	"strings"

	producer_interfaces "github.com/evolution-foundation/evolution-go/pkg/events/interfaces"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/webhooks"
)

// webhookProducer entrega por pkg/webhooks: cabeceras y firma por línea, URL
// por evento, reintentos con espera creciente y registro de entregas.
type webhookProducer struct {
	url           string // webhook global (WEBHOOK_URL)
	loggerWrapper *logger_wrapper.LoggerManager
}

func NewWebhookProducer(
	url string,
	loggerWrapper *logger_wrapper.LoggerManager,
) producer_interfaces.Producer {
	return &webhookProducer{
		url:           url,
		loggerWrapper: loggerWrapper,
	}
}

// Produce envía el evento al webhook global y al de la línea (webhookUrl).
// userID es el ID de la instancia.
func (p *webhookProducer) Produce(
	queueName string,
	payload []byte,
	webhookUrl string,
	userID string,
) error {
	if len(strings.Split(queueName, ".")) < 2 {
		return nil
	}
	svc := webhooks.Default()
	if svc == nil {
		p.loggerWrapper.GetLogger(userID).LogError("[%s] webhook service not initialized", userID)
		return nil
	}

	send := func(t webhooks.Target) {
		res := svc.Deliver(t, payload, false)
		if res.OK {
			p.loggerWrapper.GetLogger(userID).LogInfo("[%s] webhook sent - url: %s, status: %d, attempts: %d", userID, res.URL, res.StatusCode, res.Attempts)
		} else {
			p.loggerWrapper.GetLogger(userID).LogError("[%s] webhook failed after %d attempts - url: %s, error: %s", userID, res.Attempts, res.URL, res.Error)
		}
	}
	if p.url != "" {
		go send(svc.GlobalTarget(userID, p.url))
	}
	if webhookUrl != "" && webhookUrl != "disabled" {
		go send(svc.TargetFor(userID, webhookUrl))
	}
	return nil
}

// CreateGlobalQueues não faz nada para webhook producer
func (p *webhookProducer) CreateGlobalQueues() error {
	return nil
}
