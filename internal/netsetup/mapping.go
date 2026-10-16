// Package netsetup asks only the local router for a short-lived TCP mapping.
// It never consults a public address-discovery, rendezvous, or relay service.
package netsetup

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/huin/goupnp"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/jackpal/gateway"
)

const leaseSeconds = 120

type Mapping struct {
	Address, Protocol string
	renew             func(context.Context, uint32) (string, time.Duration, error)
	duration          time.Duration
	expires           time.Time
}

// Run is the text adapter retained for callers that only need notices.
func (m *Mapping) Run(ctx context.Context, report func(string)) {
	m.run(ctx, func(u Update) {
		if u.Notice != "" {
			report(u.Notice)
		}
	})
}
func Discover(ctx context.Context, port int) (*Mapping, error) {
	gw, err := gateway.DiscoverGateway()
	if err == nil && gw.To4() != nil {
		target := net.JoinHostPort(gw.String(), "5351")
		for _, kind := range []string{"PCP", "NAT-PMP"} {
			attempt, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
			m, e := udpMapping(attempt, target, port, kind)
			cancel()
			if e == nil {
				return m, nil
			}
		}
	}
	return upnpMapping(ctx, port)
}
func exchange(ctx context.Context, target string, request []byte) ([]byte, net.IP, error) {
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "udp4", target)
	if err != nil {
		return nil, nil, err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { c.Close() })
	defer stop()
	deadline := time.Now().Add(time.Second)
	if t, ok := ctx.Deadline(); ok && t.Before(deadline) {
		deadline = t
	}
	c.SetDeadline(deadline)
	local := c.LocalAddr().(*net.UDPAddr).IP
	// PCP requires the actual source address in the request header.
	if request[0] == 2 {
		copy(request[8:24], local.To16())
	}
	if _, err = c.Write(request); err != nil {
		return nil, nil, err
	}
	buf := make([]byte, 1100)
	n, err := c.Read(buf)
	return buf[:n], local, err
}
func udpMapping(ctx context.Context, target string, port int, kind string) (*Mapping, error) {
	var nonce [12]byte
	rand.Read(nonce[:])
	var external uint16 = uint16(port)
	fn := func(ctx context.Context, lifetime uint32) (string, time.Duration, error) {
		if kind == "PCP" {
			q := make([]byte, 60)
			q[0] = 2
			q[1] = 1
			binary.BigEndian.PutUint32(q[4:], lifetime)
			copy(q[24:36], nonce[:])
			q[36] = 6
			binary.BigEndian.PutUint16(q[40:], uint16(port))
			binary.BigEndian.PutUint16(q[42:], external)
			b, _, err := exchange(ctx, target, q)
			if err != nil {
				return "", 0, err
			}
			if len(b) < 60 || b[0] != 2 || b[1] != 129 || b[3] != 0 || string(b[24:36]) != string(nonce[:]) || b[36] != 6 || binary.BigEndian.Uint16(b[40:]) != uint16(port) {
				return "", 0, errors.New("invalid or rejected PCP mapping")
			}
			external = binary.BigEndian.Uint16(b[42:])
			ip := net.IP(b[44:60])
			duration := binary.BigEndian.Uint32(b[4:])
			if lifetime > 0 && (duration == 0 || external == 0 || !ip.IsGlobalUnicast()) {
				return "", 0, errors.New("unusable PCP lease")
			}
			return net.JoinHostPort(ip.String(), strconv.Itoa(int(external))), time.Duration(duration) * time.Second, nil
		}
		b, _, err := exchange(ctx, target, []byte{0, 0})
		if err != nil {
			return "", 0, err
		}
		if len(b) != 12 || b[0] != 0 || b[1] != 128 || binary.BigEndian.Uint16(b[2:]) != 0 {
			return "", 0, errors.New("invalid NAT-PMP address response")
		}
		ip := net.IP(append([]byte(nil), b[8:12]...))
		q := make([]byte, 12)
		q[1] = 2
		binary.BigEndian.PutUint16(q[4:], uint16(port))
		binary.BigEndian.PutUint16(q[6:], external)
		binary.BigEndian.PutUint32(q[8:], lifetime)
		b, _, err = exchange(ctx, target, q)
		if err != nil {
			return "", 0, err
		}
		if len(b) != 16 || b[0] != 0 || b[1] != 130 || binary.BigEndian.Uint16(b[2:]) != 0 || binary.BigEndian.Uint16(b[8:]) != uint16(port) {
			return "", 0, errors.New("invalid or rejected NAT-PMP mapping")
		}
		external = binary.BigEndian.Uint16(b[10:])
		duration := binary.BigEndian.Uint32(b[12:])
		if lifetime > 0 && (duration == 0 || external == 0 || !ip.IsGlobalUnicast()) {
			return "", 0, errors.New("unusable NAT-PMP lease")
		}
		return net.JoinHostPort(ip.String(), strconv.Itoa(int(external))), time.Duration(duration) * time.Second, nil
	}
	addr, d, err := fn(ctx, leaseSeconds)
	if err != nil {
		return nil, err
	}
	return &Mapping{Address: addr, Protocol: kind, renew: fn, duration: d, expires: time.Now().Add(d)}, nil
}

type router interface {
	GetExternalIPAddressCtx(context.Context) (string, error)
	AddPortMappingCtx(context.Context, string, uint16, string, uint16, string, bool, string, uint32) error
	DeletePortMappingCtx(context.Context, string, uint16, string) error
}

func upnpMapping(ctx context.Context, port int) (*Mapping, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type found struct {
		router  router
		service *goupnp.ServiceClient
	}
	results := make(chan found, 32)
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	publish := func(r router, s *goupnp.ServiceClient) {
		select {
		case results <- found{r, s}:
		case <-ctx.Done():
		}
	}
	wg.Add(3)
	go func() {
		defer wg.Done()
		cs, _, _ := internetgateway2.NewWANIPConnection1ClientsCtx(ctx)
		for _, c := range cs {
			publish(c, &c.ServiceClient)
		}
	}()
	go func() {
		defer wg.Done()
		cs, _, _ := internetgateway2.NewWANIPConnection2ClientsCtx(ctx)
		for _, c := range cs {
			publish(c, &c.ServiceClient)
		}
	}()
	go func() {
		defer wg.Done()
		cs, _, _ := internetgateway2.NewWANPPPConnection1ClientsCtx(ctx)
		for _, c := range cs {
			publish(c, &c.ServiceClient)
		}
	}()
	go func() { wg.Wait(); close(results) }()
	// Discovery is bounded by ctx; only the selected router gets a mapping.
	for result := range results {
		if ctx.Err() != nil {
			continue
		}
		local := result.service.LocalAddr()
		if local == nil || local.IsUnspecified() {
			if result.service.Location == nil {
				continue
			}
			host := result.service.Location.Hostname()
			if host == "" {
				continue
			}
			d := net.Dialer{}
			c, e := d.DialContext(ctx, "udp", net.JoinHostPort(host, "1900"))
			if e != nil {
				continue
			}
			local = c.LocalAddr().(*net.UDPAddr).IP
			c.Close()
		}
		m, e := mapUPnP(ctx, result.router, local.String(), port)
		if e == nil {
			return m, nil
		}
	}
	return nil, errors.New("no short-lived router mapping available")
}
func mapUPnP(ctx context.Context, r router, local string, port int) (*Mapping, error) {
	address := ""
	fn := func(ctx context.Context, lifetime uint32) (string, time.Duration, error) {
		if lifetime == 0 {
			return address, 0, r.DeletePortMappingCtx(ctx, "", uint16(port), "TCP")
		}
		ip, err := r.GetExternalIPAddressCtx(ctx)
		if err != nil {
			return "", 0, err
		}
		parsed := net.ParseIP(ip)
		if parsed == nil || !parsed.IsGlobalUnicast() {
			return "", 0, errors.New("router returned an invalid external address")
		}
		if err := r.AddPortMappingCtx(ctx, "", uint16(port), "TCP", uint16(port), local, true, "Sea Battle 2", lifetime); err != nil {
			return "", 0, fmt.Errorf("router refused a short lease: %w", err)
		}
		address = net.JoinHostPort(ip, strconv.Itoa(port))
		return address, time.Duration(lifetime) * time.Second, nil
	}
	addr, d, err := fn(ctx, leaseSeconds)
	if err != nil {
		return nil, err
	}
	return &Mapping{Address: addr, Protocol: "UPnP", renew: fn, duration: d, expires: time.Now().Add(d)}, nil
}
