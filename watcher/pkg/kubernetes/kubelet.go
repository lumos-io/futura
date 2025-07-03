package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/utils"
	"github.com/rs/zerolog/log"
)

const (
	svcAcctCACertPath   = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	svcAcctTokenPath    = "/var/run/secrets/kubernetes.io/serviceaccount/token" // #nosec
	defaultSecurePort   = "10250"
	defaultReadOnlyPort = "10255"
)

type Client interface {
	Get(path string) ([]byte, error)
}

func NewClientProvider(endpoint string, cfg *config.Configuration) (ClientProvider, error) {
	switch AuthType(cfg.Kubernetes.Auth.AuthType) {
	case AuthTypeTLS:
		return &tlsClientProvider{
			endpoint: endpoint,
			cfg:      cfg,
		}, nil
	case AuthTypeServiceAccount:
		return &saClientProvider{
			endpoint:   endpoint,
			caCertPath: svcAcctCACertPath,
			cfg:        cfg,
			tokenPath:  svcAcctTokenPath,
		}, nil
	case AuthTypeNone:
		return &readOnlyClientProvider{
			endpoint: endpoint,
		}, nil
	case AuthTypeKubeConfig:
		return &kubeConfigClientProvider{
			endpoint: endpoint,
			cfg:      cfg,
		}, nil
	default:
		return nil, fmt.Errorf("AuthType [%s] not supported", cfg.Kubernetes.Auth.AuthType)
	}
}

type ClientProvider interface {
	BuildClient() (Client, error)
}

type kubeConfigClientProvider struct {
	endpoint string
	cfg      *config.Configuration
}

func (p *kubeConfigClientProvider) BuildClient() (Client, error) {
	authConf, err := CreateRestConfig(APIConfig{
		AuthType: AuthType(p.cfg.Kubernetes.Auth.AuthType),
		Context:  p.cfg.Kubernetes.Auth.KubeContextName,
	})
	if err != nil {
		return nil, err
	}
	if p.cfg.Kubernetes.Auth.InsecureSkipVerify {
		// Override InsecureSkipVerify from kubeconfig
		authConf.CAFile = ""
		authConf.CAData = nil
		authConf.Insecure = true
	}

	client, err := rest.HTTPClientFor(authConf)
	if err != nil {
		return nil, err
	}

	joinPath, err := url.JoinPath(authConf.Host, "/api/v1/nodes/", p.endpoint, "/proxy/")
	if err != nil {
		return nil, err
	}
	return &clientImpl{
		baseURL:    joinPath,
		httpClient: *client,
		tok:        nil,
	}, nil
}

type readOnlyClientProvider struct {
	endpoint string
}

func (p *readOnlyClientProvider) BuildClient() (Client, error) {
	tr := defaultTransport()
	endpoint, err := buildEndpoint(p.endpoint, false)
	if err != nil {
		return nil, err
	}
	return &clientImpl{
		baseURL:    endpoint,
		httpClient: http.Client{Transport: tr},
		tok:        nil,
	}, nil
}

type tlsClientProvider struct {
	endpoint string
	cfg      *config.Configuration
}

func (p *tlsClientProvider) BuildClient() (Client, error) {
	rootCAs, err := systemCertPoolPlusPath(p.cfg.Kubernetes.Auth.KubeletCAFile)
	if err != nil {
		return nil, err
	}
	clientCert, err := tls.LoadX509KeyPair(p.cfg.Kubernetes.Auth.KubeletCertFile, p.cfg.Kubernetes.Auth.KubeletKeyFile)
	if err != nil {
		return nil, err
	}
	return defaultTLSClient(
		p.endpoint,
		p.cfg.Kubernetes.Auth.InsecureSkipVerify,
		rootCAs,
		[]tls.Certificate{clientCert},
		nil,
	)
}

type saClientProvider struct {
	endpoint   string
	caCertPath string
	cfg        *config.Configuration
	tokenPath  string
}

func (p *saClientProvider) BuildClient() (Client, error) {
	caCertPath := p.caCertPath
	if p.cfg.Kubernetes.Auth.KubeletCAFile != "" {
		caCertPath = p.cfg.Kubernetes.Auth.KubeletCAFile
	}
	rootCAs, err := systemCertPoolPlusPath(caCertPath)
	if err != nil {
		return nil, err
	}
	tok, err := os.ReadFile(p.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read token file %s: %w", p.tokenPath, err)
	}
	tr := defaultTransport()
	tr.TLSClientConfig = &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: p.cfg.Kubernetes.Auth.InsecureSkipVerify,
	}
	endpoint, err := buildEndpoint(p.endpoint, true)
	if err != nil {
		return nil, err
	}
	rt, err := transport.NewBearerAuthWithRefreshRoundTripper(string(tok), p.tokenPath, tr)
	if err != nil {
		return nil, err
	}

	return &clientImpl{
		baseURL: endpoint,
		httpClient: http.Client{
			Transport: rt,
		},
		tok: nil,
	}, nil
}

func defaultTLSClient(endpoint string, insecureSkipVerify bool, rootCAs *x509.CertPool, certificates []tls.Certificate, tok []byte) (*clientImpl, error) {
	tr := defaultTransport()
	tr.TLSClientConfig = &tls.Config{
		RootCAs:            rootCAs,
		Certificates:       certificates,
		InsecureSkipVerify: insecureSkipVerify,
	}
	endpoint, err := buildEndpoint(endpoint, true)
	if err != nil {
		return nil, err
	}
	return &clientImpl{
		baseURL:    endpoint,
		httpClient: http.Client{Transport: tr},
		tok:        tok,
	}, nil
}

// buildEndpoint builds a kubelet endpoint based on value provided by user and whether secure or read-only endpoint
// should be used.
func buildEndpoint(endpoint string, useSecurePort bool) (string, error) {
	if endpoint == "" {
		// This will work if hostNetwork is turned on, in which case the pod has access
		// to the node's loopback device.
		// https://kubernetes.io/docs/concepts/policy/pod-security-policy/#host-namespaces
		host, err := os.Hostname()
		if err != nil {
			return "", fmt.Errorf("unable to get hostname for default endpoint: %w", err)
		}

		if useSecurePort {
			endpoint = fmt.Sprintf("https://%s:%s", host, defaultSecurePort)
		} else {
			endpoint = fmt.Sprintf("http://%s:%s", host, defaultReadOnlyPort)
		}
		log.Logger.Warn().Msgf("Kubelet endpoint not defined, using default endpoint %s", endpoint)
		return endpoint, nil
	}

	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if useSecurePort {
			return "https://" + endpoint, nil
		}
		return "http://" + endpoint, nil
	}

	return endpoint, nil
}

func defaultTransport() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}

// clientImpl

var _ Client = (*clientImpl)(nil)

type clientImpl struct {
	baseURL    string
	httpClient http.Client
	tok        []byte
}

func (c *clientImpl) Get(path string) ([]byte, error) {
	req, err := c.buildReq(path)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			log.Logger.Warn().Err(closeErr).Msg("failed to close response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Kubelet response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kubelet request GET %s failed - %q, response: %q",
			utils.URL(req.URL), resp.Status, string(body))
	}

	return body, nil
}

func (c *clientImpl) buildReq(p string) (*http.Request, error) {
	reqURL, err := url.JoinPath(c.baseURL, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.tok != nil {
		req.Header.Set("Authorization", fmt.Sprintf("bearer %s", c.tok))
	}
	return req, nil
}

func systemCertPoolPlusPath(certPath string) (*x509.CertPool, error) {
	var sysCerts *x509.CertPool
	var err error
	if runtime.GOOS == "windows" {
		sysCerts, err = x509.NewCertPool(), nil
	} else {
		sysCerts, err = x509.SystemCertPool()
	}
	if err != nil {
		return nil, fmt.Errorf("could not load system x509 cert pool: %w", err)
	}
	return certPoolPlusPath(sysCerts, certPath)
}

func certPoolPlusPath(certPool *x509.CertPool, certPath string) (*x509.CertPool, error) {
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("cert path %s could not be read: %w", certPath, err)
	}
	ok := certPool.AppendCertsFromPEM(certBytes)
	if !ok {
		return nil, errors.New("AppendCertsFromPEM failed")
	}
	return certPool, nil
}
