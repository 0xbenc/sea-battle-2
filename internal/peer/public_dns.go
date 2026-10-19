package peer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Newly allocated Quick Tunnel names can remain negatively cached in a local
// resolver. Only those names get a bounded fallback to Cloudflare/Google HTTPS DNS
// services. No OS DNS settings change; both outer HTTPS and inner TLS pins remain
// verified, and the DNS query never contains invitation credentials.
func quickTunnelHost(host string) bool {
	host = strings.ToLower(host)
	const suffix = ".trycloudflare.com"
	if !strings.HasSuffix(host, suffix) {
		return false
	}
	label := strings.TrimSuffix(host, suffix)
	if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, r := range label {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}
func dialPublicWebSocket(ctx context.Context, config *websocket.Config) (*websocket.Conn, error) {
	initial := ctx
	stop := func() {}
	if quickTunnelHost(config.Location.Hostname()) {
		initial, stop = context.WithTimeout(ctx, 2*time.Second)
	}
	ws, err := config.DialContext(initial)
	stop()
	if err == nil {
		return ws, nil
	}
	cause := err
	if e, ok := err.(*websocket.DialError); ok {
		cause = e.Err
	}
	var dns *net.DNSError
	if !quickTunnelHost(config.Location.Hostname()) || ctx.Err() != nil || (!errors.As(cause, &dns) && !errors.Is(cause, context.DeadlineExceeded)) {
		return nil, err
	}
	addresses, resolveErr := racePublicDNS(ctx, http.DefaultClient, []string{"https://cloudflare-dns.com/dns-query", "https://dns.google/resolve"}, config.Location.Hostname())
	if resolveErr != nil {
		return nil, fmt.Errorf("system DNS: %v; public DNS: %w", err, resolveErr)
	}
	for _, ip := range addresses {
		attempt, cancel := context.WithTimeout(ctx, 2*time.Second)
		ws, resolveErr = websocketAtIP(attempt, config, ip)
		cancel()
		if resolveErr == nil {
			return ws, nil
		}
	}
	return nil, fmt.Errorf("public DNS connection: %w", resolveErr)
}
func resolvePublicDNS(ctx context.Context, client *http.Client, endpoint, host string) ([]net.IP, error) {
	if !quickTunnelHost(host) {
		return nil, errors.New("public DNS fallback is limited to Quick Tunnel names")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?name="+url.QueryEscape(host)+"&type=A", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-json")
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DNS HTTPS status %d", response.StatusCode)
	}
	var answer struct {
		Status int
		Answer []struct {
			Name string
			Type int
			Data string
		}
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&answer); err != nil {
		return nil, err
	}
	var ips []net.IP
	if answer.Status == 0 {
		for _, record := range answer.Answer {
			if record.Type != 1 || !strings.EqualFold(strings.TrimSuffix(record.Name, "."), host) {
				continue
			}
			ip := net.ParseIP(record.Data)
			if ip != nil && ip.To4() != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
				ips = append(ips, ip)
				if len(ips) == 2 {
					break
				}
			}
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("public DNS returned no usable address")
	}
	return ips, nil
}
func websocketAtIP(ctx context.Context, config *websocket.Config, ip net.IP) (*websocket.Conn, error) {
	port := config.Location.Port()
	if port == "" {
		port = "443"
	}
	// Keep the original hostname for SNI, certificate validation and HTTP Host.
	d := tls.Dialer{Config: &tls.Config{ServerName: config.Location.Hostname(), MinVersion: tls.VersionTLS12}}
	raw, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { abortConn(raw) })
	ws, err := websocket.NewClient(config, raw)
	stopped := stop()
	if !stopped {
		abortConn(raw)
		return nil, ctx.Err()
	}
	if err != nil {
		abortConn(raw)
		return nil, err
	}
	return ws, nil
}

// CheckPublicEndpoint verifies the freshly allocated HTTPS/WebSocket endpoint is
// reachable before the host offers its invitation. It never sends a game key.
func CheckPublicEndpoint(ctx context.Context, address string) error {
	if err := ValidAddress(address); err != nil {
		return err
	}
	config, err := websocket.NewConfig(address, "https://sea-battle.invalid")
	if err != nil {
		return err
	}
	ws, err := dialPublicWebSocket(ctx, config)
	if err != nil {
		return err
	}
	ws.SetDeadline(time.Now())
	ws.Close()
	return nil
}

func racePublicDNS(ctx context.Context, client *http.Client, endpoints []string, host string) ([]net.IP, error) {
	ctx, cancel := context.WithCancel(ctx)
	type result struct {
		ips []net.IP
		err error
	}
	results := make(chan result, len(endpoints))
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			ips, err := resolvePublicDNS(ctx, client, endpoint, host)
			results <- result{ips, err}
		}(endpoint)
	}
	var failures []error
	for range endpoints {
		select {
		case r := <-results:
			if r.err == nil {
				return r.ips, nil
			}
			failures = append(failures, r.err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errors.Join(failures...)
}
