// Package peer implements pinned TLS over direct TCP or optional WebSocket tunnels.
package peer

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const Version = 1
const MaxInvitation = 8192

// The invitation is a bearer credential. Never send it to logs or a public service.
type Invitation struct {
	Version     int      `json:"v"`
	Addresses   []string `json:"a"`
	Fingerprint string   `json:"f"`
	Token       string   `json:"k"`
	Expires     int64    `json:"e"`
}

func (i Invitation) Encode() (string, error) {
	if err := i.Validate(); err != nil {
		return "", err
	}
	b, e := json.Marshal(i)
	return "sb2:" + base64.RawURLEncoding.EncodeToString(b), e
}
func ParseInvitation(s string) (Invitation, error) {
	var i Invitation
	if len(s) > MaxInvitation {
		return i, errors.New("invitation is too long")
	}
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	if !strings.HasPrefix(s, "sb2:") {
		return i, errors.New("paste the complete invitation beginning with sb2:")
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, "sb2:"))
	if e != nil {
		return i, errors.New("invitation is damaged or incomplete")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&i); e != nil {
		return i, errors.New("invalid invitation data")
	}
	if d.Decode(new(any)) != io.EOF {
		return i, errors.New("trailing invitation data")
	}
	return i, i.Validate()
}
func (i Invitation) Validate() error {
	if i.Version != Version {
		return errors.New("invitation uses an unsupported protocol version")
	}
	if time.Now().Unix() > i.Expires {
		return errors.New("invitation expired; ask the host for a new one")
	}
	if i.Expires > time.Now().Add(48*time.Hour).Unix() {
		return errors.New("invalid invitation expiry")
	}
	if b, e := hex.DecodeString(i.Fingerprint); e != nil || len(b) != 32 {
		return errors.New("invalid host fingerprint")
	}
	if b, e := base64.RawURLEncoding.DecodeString(i.Token); e != nil || len(b) != 32 {
		return errors.New("invalid session key")
	}
	if len(i.Addresses) == 0 || len(i.Addresses) > 16 {
		return errors.New("invitation needs 1-16 addresses")
	}
	for _, a := range i.Addresses {
		if err := ValidAddress(a); err != nil {
			return err
		}
	}
	return nil
}
func ValidAddress(a string) error {
	if len(a) > 300 {
		return errors.New("address too long")
	}
	if strings.HasPrefix(a, "wss://") {
		return validWebSocketAddress(a)
	}
	host, port, e := net.SplitHostPort(a)
	if e != nil || host == "" {
		return errors.New("address must be host:port (IPv6: [address]:port)")
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1 || n > 65535 {
		return errors.New("port must be 1-65535")
	}
	for _, r := range host {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(".-:%_", r)) {
			return errors.New("invalid address character")
		}
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsUnspecified() || ip.IsMulticast()) {
		return errors.New("use a reachable unicast address")
	}
	return nil
}
func secret() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func certificate() (tls.Certificate, string, error) {
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		return tls.Certificate{}, "", e
	}
	serial, e := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if e != nil {
		return tls.Certificate{}, "", e
	}
	tpl := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Sea Battle 2 direct host"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(25 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, e := x509.CreateCertificate(rand.Reader, tpl, tpl, pub, priv)
	if e != nil {
		return tls.Certificate{}, "", e
	}
	key, e := x509.MarshalPKCS8PrivateKey(priv)
	if e != nil {
		return tls.Certificate{}, "", e
	}
	cert, e := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
	sum := sha256.Sum256(der)
	return cert, hex.EncodeToString(sum[:]), e
}
func pinnedConfig(fingerprint string) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: true, // Trust is the exact certificate fingerprint shared out of band.
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) != 1 {
				return errors.New("unexpected host certificate chain")
			}
			sum := sha256.Sum256(cs.PeerCertificates[0].Raw)
			if hex.EncodeToString(sum[:]) != fingerprint {
				return errors.New("host fingerprint mismatch; invitation does not match this server")
			}
			return nil
		}}
}
func LocalAddresses(port int) []string {
	var out []string
	ifs, _ := net.Interfaces()
	for _, iface := range ifs {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		as, _ := iface.Addrs()
		for _, a := range as {
			ip, _, e := net.ParseCIDR(a.String())
			if e != nil || !ip.IsGlobalUnicast() || ip.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, net.JoinHostPort(ip.String(), strconv.Itoa(port)))
			if len(out) == 12 {
				return out
			}
		}
	}
	if len(out) == 0 {
		out = []string{net.JoinHostPort("127.0.0.1", strconv.Itoa(port))}
	}
	return out
}
func ConnectionHelp(port int) string {
	return fmt.Sprintf("Allow inbound TCP %d on the host. On different networks, try a shared VPN such as Tailscale (running inside WSL when hosting there). If automatic mapping is unavailable, forward TCP %d to this computer and add the router's public address. With upstream NAT/CGNAT, try the other player hosting or reachable IPv6. For a public tunnel, host with python3 tools/host-public.py.", port, port)
}
