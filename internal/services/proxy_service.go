package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type ProxyService struct {
	client   *http.Client
	counters map[string]*atomic.Uint64
}

func NewProxyService() *ProxyService {
	return &ProxyService{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		counters: make(map[string]*atomic.Uint64),
	}
}

func (p *ProxyService) Forward(ctx context.Context, backendURLs []string, method, originalPath, routePath string, headers http.Header, body []byte, timeoutMs int) (*http.Response, error) {
	if len(backendURLs) == 0 {
		return nil, fmt.Errorf("no backend URLs configured")
	}

	backendURL := p.selectBackend(backendURLs, originalPath)

	trimmed := originalPath
	if strings.HasPrefix(originalPath, routePath) {
		trimmed = strings.TrimPrefix(originalPath, routePath)
	}
	if trimmed == "" {
		trimmed = "/"
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutMs) * time.Millisecond,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	finalURL := backendURL + trimmed

	req, err := http.NewRequestWithContext(ctx, method, finalURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (p *ProxyService) selectBackend(backends []string, path string) string {
	if len(backends) == 1 {
		return backends[0]
	}

	counter, ok := p.counters[path]
	if !ok {
		counter = &atomic.Uint64{}
		p.counters[path] = counter
	}

	index := counter.Add(1) % uint64(len(backends))
	return backends[index]
}

func readBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return body, nil
}
