package peer

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sea-battle-2/internal/game"
	"sync"
	"time"
)

const maxMessage = 64 * 1024

type Command struct {
	Type     string           `json:"type"`
	Resume   string           `json:"resume,omitempty"`
	Token    string           `json:"token,omitempty"`
	Version  int              `json:"version,omitempty"`
	Fleet    []game.Placement `json:"fleet,omitempty"`
	At       game.Coord       `json:"at"`
	Expected int              `json:"expected"`
	Match    int              `json:"match"`
}
type Message struct {
	Invitation *Invitation `json:"invitation,omitempty"`
	Type       string      `json:"type"`
	View       *game.View  `json:"view,omitempty"`
	Error      string      `json:"error,omitempty"`
}
type subscriber struct {
	conn   net.Conn
	resume string
	out    chan Message
}
type Server struct {
	listener          net.Listener
	mu                sync.Mutex
	game              *game.Game
	peers             [2]*subscriber
	hostKey, guestKey string
	fingerprint       string
	expires           int64
	done              chan struct{}
	closeOnce         sync.Once
	wg                sync.WaitGroup
	admission         chan struct{}
	connections       map[net.Conn]struct{}
	invitation        *Invitation
}

func Host(address string) (*Server, error) {
	cert, fp, e := certificate()
	if e != nil {
		return nil, e
	}
	hk, e := secret()
	if e != nil {
		return nil, e
	}
	gk, e := secret()
	if e != nil {
		return nil, e
	}
	l, e := tls.Listen("tcp", address, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13})
	if e != nil {
		return nil, e
	}
	var coin [1]byte
	rand.Read(coin[:])
	s := &Server{listener: l, game: game.New(int(coin[0] % 2)), hostKey: hk, guestKey: gk, fingerprint: fp, expires: time.Now().Add(24 * time.Hour).Unix(), done: make(chan struct{}), admission: make(chan struct{}, 16), connections: make(map[net.Conn]struct{})}
	s.wg.Add(1)
	go s.accept()
	return s, nil
}
func (s *Server) Port() int { return s.listener.Addr().(*net.TCPAddr).Port }
func (s *Server) Invite(addresses []string) (Invitation, error) {
	i := Invitation{Version: Version, Addresses: append([]string(nil), addresses...), Fingerprint: s.fingerprint, Token: s.guestKey, Expires: s.expires}
	return i, i.Validate()
}
func (s *Server) LocalInvite() Invitation {
	return Invitation{Version: Version, Addresses: []string{fmt.Sprintf("127.0.0.1:%d", s.Port())}, Fingerprint: s.fingerprint, Token: s.hostKey, Expires: s.expires}
}
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		s.listener.Close()
		s.mu.Lock()
		for c := range s.connections {
			abortConn(c)
		}
		s.mu.Unlock()
		s.wg.Wait()
	})
}
func (s *Server) accept() {
	defer s.wg.Done()
	for {
		c, e := s.listener.Accept()
		if e != nil {
			return
		}
		select {
		case s.admission <- struct{}{}:
			s.mu.Lock()
			select {
			case <-s.done:
				s.mu.Unlock()
				abortConn(c)
				<-s.admission
				return
			default:
			}
			s.connections[c] = struct{}{}
			s.mu.Unlock()
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer func() { s.mu.Lock(); delete(s.connections, c); s.mu.Unlock(); <-s.admission }()
				s.serve(c)
			}()
		default:
			abortConn(c)
		}
	}
}
func (s *Server) serve(c net.Conn) {
	defer abortConn(c)
	c.SetDeadline(time.Now().Add(5 * time.Second))
	if tc, ok := c.(*tls.Conn); ok {
		if e := tc.HandshakeContext(context.Background()); e != nil {
			return
		}
	}
	scanner := bufio.NewScanner(c)
	scanner.Buffer(make([]byte, 4096), maxMessage)
	if !scanner.Scan() {
		return
	}
	hello, decodeErr := decodeCommand(scanner.Bytes())
	if decodeErr != nil || hello.Type != "hello" || hello.Version != Version || len(hello.Resume) > 128 {
		return
	}
	player := -1
	if subtle.ConstantTimeCompare([]byte(hello.Token), []byte(s.hostKey)) == 1 {
		player = 0
	}
	if subtle.ConstantTimeCompare([]byte(hello.Token), []byte(s.guestKey)) == 1 {
		player = 1
	}
	if player < 0 || time.Now().Unix() > s.expires {
		json.NewEncoder(c).Encode(Message{Type: "fatal", Error: "Invitation key rejected or expired. Ask the host for the current invitation."})
		return
	}
	sub := &subscriber{conn: c, resume: hello.Resume, out: make(chan Message, 8)}
	s.mu.Lock()
	if old := s.peers[player]; old != nil {
		// A reconnect from this same client instance may replace a half-open socket.
		// A second independent client with the same invite still cannot take its seat.
		if hello.Resume != "" && subtle.ConstantTimeCompare([]byte(hello.Resume), []byte(old.resume)) == 1 {
			abortConn(old.conn)
		} else {
			s.mu.Unlock()
			json.NewEncoder(c).Encode(Message{Type: "error", Error: "this player is already connected"})
			return
		}
	}

	select {
	case <-s.done:
		s.mu.Unlock()
		return
	default:
	}
	s.peers[player] = sub
	if player == 1 && s.invitation != nil {
		s.sendLocked(sub, Message{Type: "invitation", Invitation: s.invitation})
	}
	s.broadcastLocked()
	s.mu.Unlock()
	c.SetDeadline(time.Time{})
	writeDone := make(chan struct{})
	stopWriter := make(chan struct{})
	go func() {
		defer close(writeDone)
		enc := json.NewEncoder(c)
		for {
			select {
			case m := <-sub.out:
				c.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if enc.Encode(m) != nil {
					abortConn(c)
					return
				}
			case <-s.done:
				return
			case <-stopWriter:
				return
			}
		}
	}()
	// Reader owns the connection lifetime; closing it wakes both IO directions.
	for {
		c.SetReadDeadline(time.Now().Add(35 * time.Second))
		if !scanner.Scan() {
			break
		}
		cmd, decodeErr := decodeCommand(scanner.Bytes())
		if decodeErr != nil {
			break
		}
		if cmd.Type == "ping" {
			s.mu.Lock()
			s.sendLocked(sub, Message{Type: "pong"})
			s.mu.Unlock()
			continue
		}
		s.mu.Lock()
		if s.peers[player] != sub {
			s.mu.Unlock()
			break
		}
		var err error
		if cmd.Match != s.game.Match {
			err = errors.New("match changed; wait for the current board")
		} else {
			switch cmd.Type {
			case "ready":
				err = s.game.SetFleet(player, cmd.Fleet)
			case "fire":
				if s.peers[0] == nil || s.peers[1] == nil {
					err = errors.New("opponent disconnected; battle is paused")
				} else {
					err = s.game.Fire(player, cmd.At, cmd.Expected)
				}
			case "rematch":
				err = s.game.RequestRematch(player)
			default:
				err = errors.New("unknown command")
			}
		}
		if err != nil {
			s.sendLocked(sub, Message{Type: "error", Error: err.Error()})
			s.sendViewLocked(player)
		} else {
			s.broadcastLocked()
		}
		s.mu.Unlock()
	}
	s.mu.Lock()
	if s.peers[player] == sub {
		s.peers[player] = nil
		s.broadcastLocked()
	}
	s.mu.Unlock()
	abortConn(c)
	// The writer also selects its explicit stop channel, below.
	close(stopWriter)
	<-writeDone
}
func (s *Server) sendLocked(p *subscriber, m Message) {
	select {
	case p.out <- m:
	default:
		abortConn(p.conn)
	}
}
func (s *Server) sendViewLocked(player int) {
	p := s.peers[player]
	if p == nil {
		return
	}
	v := s.game.View(player)
	v.Connected = [2]bool{s.peers[0] != nil, s.peers[1] != nil}
	s.sendLocked(p, Message{Type: "state", View: &v})
}
func (s *Server) broadcastLocked() { s.sendViewLocked(0); s.sendViewLocked(1) }

// UpdateAddresses publishes new routes over the existing authenticated guest
// connection. If every path is already gone, a fresh invite must be shared out of band.
func (s *Server) UpdateAddresses(addresses []string) (Invitation, error) {
	i, err := s.Invite(addresses)
	if err != nil {
		return i, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := i
	stored.Addresses = append([]string(nil), i.Addresses...)
	s.invitation = &stored
	if p := s.peers[1]; p != nil {
		s.sendLocked(p, Message{Type: "invitation", Invitation: &stored})
	}
	return i, nil
}
