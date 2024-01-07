package webhook

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/event"
	"github.com/opisvigilant/futura/watcher/internal/logger"
)

// Webhook handler implements handler.Handler interface,
// Notify event to Webhook
type Webhook struct {
	URL string
}

// WebhookMessage for messages
type WebhookMessage struct {
	EventMeta EventMeta `json:"eventmeta"`
	Text      string    `json:"text"`
	Time      time.Time `json:"time"`
}

// EventMeta containes the meta data about the event occurred
type EventMeta struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Reason    string `json:"reason"`
}

// Init prepares Webhook configuration
func (m *Webhook) Init(c *config.Configuration) error {
	if c.Handler.Webhook.URL == "" {
		return fmt.Errorf("missing url for webhook")
	}

	m.URL = c.Handler.Webhook.URL
	cert := c.Handler.Webhook.Cert
	tlsSkip := c.Handler.Webhook.TlsSkip

	if tlsSkip {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		if cert == "" {
			return fmt.Errorf("no webhook cert is given")
		} else {
			caCert, err := os.ReadFile(cert)
			if err != nil {
				return err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: caCertPool}
		}
	}
	return nil
}

// Handle handles an event.
func (m *Webhook) Handle(e event.Event) {
	webhookMessage := &WebhookMessage{
		EventMeta: EventMeta{
			Kind:      e.Kind,
			Name:      e.Name,
			Namespace: e.Namespace,
			Reason:    e.Reason,
		},
		Text: e.Message(),
		Time: time.Now(),
	}

	if err := postMessage(m.URL, webhookMessage); err != nil {
		logger.Logger().Err(err).Msg("failed to post to webhook URL")
		return
	}
	logger.Logger().Info().Msgf("Message successfully sent to %s at %s ", m.URL, time.Now())
}

func postMessage(url string, webhookMessage *WebhookMessage) error {
	message, err := json.Marshal(webhookMessage)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}
