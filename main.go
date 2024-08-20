package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"http-proxier/pkg/proxy"
)

func main() {
	// Parse the target URL for the reverse proxy
	target, err := url.Parse("https://www.cnn.com")
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	// Create a logger
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.SetLevel(logrus.InfoLevel)

	// Configure the proxy
	config := proxy.ProxyConfig{
		TargetURL:           target,
		InsecureSkipVerify:  true, // Only for development; disable in production!
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
		Logger:              logger,
	}

	// Create the reverse proxy
	reverseProxy := proxy.NewReverseProxy(config)

	// Customize the director to copy all headers
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Ensure the Host header is correctly set
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", config.TargetURL.Host)
		req.Host = config.TargetURL.Host

		// Copy headers from the original request to the outgoing request
		for key, values := range req.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Handle the redirect responses manually
	reverseProxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			// Handle the redirect: change the location header to the proxy's address
			redirectURL, err := resp.Location()
			if err == nil {
				resp.Header.Set("Location", config.TargetURL.ResolveReference(redirectURL).String())
			}
		}
		return nil
	}

	// Set up the HTTP server to handle incoming requests and proxy them
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reverseProxy.ServeHTTP(w, r)
	})

	// Start the server
	port := ":8080"
	logger.Infof("Starting proxy server on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
