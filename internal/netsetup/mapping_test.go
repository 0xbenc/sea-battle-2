package netsetup

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestRouterUDPLeases(t *testing.T) {
	for _, kind := range []string{"PCP", "NAT-PMP"} {
		t.Run(kind, func(t *testing.T) {
			c, err := net.ListenPacket("udp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			leases := make(chan uint32, 8)
			done := make(chan struct{})
			go func() {
				defer close(done)
				buf := make([]byte, 1024)
				for {
					n, a, e := c.ReadFrom(buf)
					if e != nil {
						return
					}
					q := append([]byte(nil), buf[:n]...)
					var b []byte
					if kind == "PCP" {
						b = make([]byte, 60)
						b[0] = 2
						b[1] = 129
						copy(b[4:8], q[4:8])
						copy(b[24:44], q[24:44])
						copy(b[44:], net.ParseIP("203.0.113.8").To16())
						leases <- binary.BigEndian.Uint32(q[4:])
					} else if n == 2 {
						b = make([]byte, 12)
						b[1] = 128
						copy(b[8:], net.ParseIP("203.0.113.8").To4())
					} else {
						b = make([]byte, 16)
						b[1] = 130
						copy(b[8:12], q[4:8])
						copy(b[12:], q[8:])
						leases <- binary.BigEndian.Uint32(q[8:])
					}
					c.WriteTo(b, a)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			m, err := udpMapping(ctx, c.LocalAddr().String(), 4747, kind)
			if err != nil {
				t.Fatal(err)
			}
			if m.Address != "203.0.113.8:4747" || m.duration != 120*time.Second {
				t.Fatalf("%+v", m)
			}
			if <-leases != 120 {
				t.Fatal("lease is not short")
			}
			if _, _, err = m.renew(ctx, 120); err != nil {
				t.Fatal(err)
			}
			if <-leases != 120 {
				t.Fatal("renew failed")
			}
			stop, cancelStop := context.WithCancel(context.Background())
			cancelStop()
			m.Run(stop, func(s string) { t.Error(s) })
			if <-leases != 0 {
				t.Fatal("lease not removed")
			}
			c.Close()
			<-done
		})
	}
}
func TestPCPRejectsMismatchedNonce(t *testing.T) {
	c, _ := net.ListenPacket("udp4", "127.0.0.1:0")
	defer c.Close()
	go func() {
		b := make([]byte, 100)
		_, a, err := c.ReadFrom(b)
		if err != nil {
			return
		}
		b[1] = 129
		b[24] ^= 255
		c.WriteTo(b[:60], a)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := udpMapping(ctx, c.LocalAddr().String(), 4747, "PCP"); err == nil {
		t.Fatal("accepted forged nonce")
	}
}

type fakeRouter struct {
	mu        sync.Mutex
	lifetimes []uint32
	deleted   bool
	refuse    bool
	ip        string
}

func (f *fakeRouter) GetExternalIPAddressCtx(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ip != "" {
		return f.ip, nil
	}
	return "203.0.113.9", nil
}
func (f *fakeRouter) AddPortMappingCtx(_ context.Context, _ string, external uint16, protocol string, internal uint16, local string, enabled bool, desc string, lifetime uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lifetimes = append(f.lifetimes, lifetime)
	if f.refuse {
		return errors.New("permanent mappings only")
	}
	if protocol != "TCP" || !enabled || external != internal || local == "" {
		return errors.New("wrong parameters")
	}
	return nil
}
func (f *fakeRouter) DeletePortMappingCtx(context.Context, string, uint16, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = true
	return nil
}
func TestUPnPShortLeaseOnly(t *testing.T) {
	f := &fakeRouter{}
	m, err := mapUPnP(context.Background(), f, "192.168.1.9", 4747)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.Run(ctx, func(string) {})
	if !f.deleted || len(f.lifetimes) != 1 || f.lifetimes[0] != 120 {
		t.Fatal("lease cleanup")
	}
	f = &fakeRouter{refuse: true}
	if _, err = mapUPnP(context.Background(), f, "192.168.1.9", 4747); err == nil {
		t.Fatal("must fail instead of permanent mapping")
	}
	if len(f.lifetimes) != 1 || f.lifetimes[0] == 0 {
		t.Fatal("attempted permanent mapping")
	}
}

func TestCancellationInterruptsRouterRead(t *testing.T) {
	c, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	entered := make(chan struct{})
	go func() { buf := make([]byte, 100); c.ReadFrom(buf); close(entered) }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); exchange(ctx, c.LocalAddr().String(), []byte{0, 0}) }()
	<-entered
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("router read ignored cancellation")
	}
}

func TestUPnPRenewalRefreshesWANAddress(t *testing.T) {
	f := &fakeRouter{}
	m, err := mapUPnP(context.Background(), f, "192.168.1.9", 4747)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.ip = "203.0.113.10"
	f.mu.Unlock()
	address, _, err := m.renew(context.Background(), leaseSeconds)
	if err != nil {
		t.Fatal(err)
	}
	if address != "203.0.113.10:4747" {
		t.Fatal("renewal retained stale WAN address", address)
	}
}
