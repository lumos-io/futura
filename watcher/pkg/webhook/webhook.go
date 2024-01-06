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

	"github.com/opisvigilant/futura/pkg/logger"
	"github.com/opisvigilant/futura/watcher/pkg/event"
)

// TODO: fix the below
var webhookErrMsg = `
%s

You need to set Webhook url, and Webhook cert if you use self signed certificates,
using "--url/-u" and "--cert", or using environment variables:

export KW_WEBHOOK_URL=webhook_url
export KW_WEBHOOK_CERT=/path/of/cert

Command line flags will override environment variables

`

// Webhook handler implements handler.Handler interface,
// Notify event to Webhook channel
type Webhook struct {
	Url string
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
func (m *Webhook) Init(url, cert string, tlsSkip bool) error {
	if url == "" {
		url = os.Getenv("KW_WEBHOOK_URL")
	}
	if cert == "" {
		cert = os.Getenv("KW_WEBHOOK_CERT")
	}

	m.Url = url

	if tlsSkip {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		if cert == "" {
			logger.Logger().Info().Msg("No webhook cert is given")
		} else {
			caCert, err := os.ReadFile(cert)
			if err != nil {
				logger.Logger().Err(err).Msg("")
				return err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: caCertPool}
		}
	}

	return checkMissingWebhookVars(m)
}

// Handle handles an event.
func (m *Webhook) Handle(e event.Event) {
	webhookMessage := prepareWebhookMessage(e, m)

	err := postMessage(m.Url, webhookMessage)
	if err != nil {
		logger.Logger().Err(err).Msg("")
		return
	}

	logger.Logger().Info().Msgf("Message successfully sent to %s at %s ", m.Url, time.Now())
}

func checkMissingWebhookVars(s *Webhook) error {
	if s.Url == "" {
		return fmt.Errorf(webhookErrMsg, "Missing Webhook url")
	}

	return nil
}

func prepareWebhookMessage(e event.Event, m *Webhook) *WebhookMessage {
	return &WebhookMessage{
		EventMeta: EventMeta{
			Kind:      e.Kind,
			Name:      e.Name,
			Namespace: e.Namespace,
			Reason:    e.Reason,
		},
		Text: e.Message(),
		Time: time.Now(),
	}
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
