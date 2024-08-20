package proxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

type ProxyConfig struct {
	TargetURL           *url.URL
	InsecureSkipVerify  bool
	MaxIdleConns        int
	IdleConnTimeout     time.Duration
	DisableCompression  bool
	TLSHandshakeTimeout time.Duration
	Logger              *logrus.Logger
}

func NewReverseProxy(config ProxyConfig) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(config.TargetURL)

	// Customize the director to modify the request before it's sent to the backend
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", config.TargetURL.Host)
		req.Host = config.TargetURL.Host
	}

	// Customize the transport for more control over the connection
	proxy.Transport = &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: config.InsecureSkipVerify}, // Only use this in development!
		MaxIdleConns:        config.MaxIdleConns,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableCompression:  config.DisableCompression,
		TLSHandshakeTimeout: config.TLSHandshakeTimeout,
	}

	// Customize error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if config.Logger != nil {
			config.Logger.Errorf("Proxy error: %v", err)
		}
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Proxy Error"))
	}

	return proxy
}
