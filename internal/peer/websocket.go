package peer

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/websocket"
)

// WebSocketHandler carries the existing pinned TLS protocol inside WebSocket.
// A public HTTP tunnel can forward this endpoint without learning game credentials
// or fleet data. The tunnel is optional and managed separately by the host.
func (s *Server) WebSocketHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sb2" {
			http.NotFound(w, r)
			return
		}
		select {
		case s.admission <- struct{}{}:
		default:
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		defer func() { <-s.admission }()
		websocket.Server{
			Handshake: func(*websocket.Config, *http.Request) error { return nil },
			Handler: func(ws *websocket.Conn) {
				ws.PayloadType = websocket.BinaryFrame
				conn := tls.Server(ws, s.tlsConfig)
				s.mu.Lock()
				select {
				case <-s.done:
					s.mu.Unlock()
					abortConn(conn)
					return
				default:
				}
				s.connections[conn] = struct{}{}
				s.wg.Add(1)
				s.mu.Unlock()
				defer s.wg.Done()
				defer func() { s.mu.Lock(); delete(s.connections, conn); s.mu.Unlock() }()
				s.serve(conn)
			},
		}.ServeHTTP(w, r)
	})
}

func validWebSocketAddress(a string) error {
	u, err := url.Parse(a)
	if err != nil || u.Scheme != "wss" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/sb2" {
		return errors.New("tunnel address must be wss://host[:port]/sb2 without credentials, query or fragment")
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return ValidAddress(net.JoinHostPort(host, port))
}

func dialPinned(ctx context.Context, addr, fingerprint string) (net.Conn, error) {
	if !strings.HasPrefix(addr, "wss://") {
		d := tls.Dialer{NetDialer: &net.Dialer{}, Config: pinnedConfig(fingerprint)}
		return d.DialContext(ctx, "tcp", addr)
	}
	return dialWebSocket(ctx, addr, fingerprint)
}
func dialWebSocket(ctx context.Context, addr, fingerprint string) (net.Conn, error) {
	config, err := websocket.NewConfig(addr, "https://sea-battle.invalid")
	if err != nil {
		return nil, err
	}
	ws, err := dialPublicWebSocket(ctx, config)
	if err != nil {
		return nil, err
	}
	ws.PayloadType = websocket.BinaryFrame
	conn := tls.Client(ws, pinnedConfig(fingerprint))
	if err = conn.HandshakeContext(ctx); err != nil {
		abortConn(conn)
		return nil, err
	}
	return conn, nil
}
