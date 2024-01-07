package webhook

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/opisvigilant/futura/watcher/internal/config"
)

func TestWebhookInit(t *testing.T) {
	s := &Webhook{}
	expectedError := fmt.Errorf("missing url for webhook")

	var Tests = []struct {
		config *config.Configuration
		err    error
	}{
		{&config.Configuration{
			Handler: &config.Handler{
				Webhook: &config.Webhook{
					URL:     "foo",
					Cert:    "",
					TlsSkip: true,
				},
			},
		}, nil},
		{&config.Configuration{
			Handler: &config.Handler{
				Webhook: &config.Webhook{
					URL: "",
				},
			},
		}, expectedError},
	}

	for _, tt := range Tests {
		if err := s.Init(tt.config); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("Init(): %v", err)
		}
	}
}
