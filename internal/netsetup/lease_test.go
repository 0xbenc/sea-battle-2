package netsetup

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestLostRenewalReplyPreservesRouteAndRecoversNewEndpoint(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var mu sync.Mutex
		calls, deletes := 0, 0
		m := &Mapping{Address: "203.0.113.1:4747", Protocol: "PCP", duration: 120 * time.Second}
		m.renew = func(ctx context.Context, life uint32) (string, time.Duration, error) {
			mu.Lock()
			defer mu.Unlock()
			if life == 0 {
				deletes++
				return "", 0, nil
			}
			calls++
			if calls == 1 {
				return "", 0, errors.New("lost response")
			}
			return "203.0.113.2:5555", 120 * time.Second, nil
		}
		updates := make(chan Update, 8)
		done := make(chan struct{})
		go func() { defer close(done); m.run(ctx, func(u Update) { updates <- u }) }()
		if u := <-updates; u.State != "active" {
			t.Fatal(u)
		}
		time.Sleep(60 * time.Second)
		synctest.Wait()
		u := <-updates
		if u.State != "retrying" || u.Address != "203.0.113.1:4747" {
			t.Fatal(u)
		}
		mu.Lock()
		if deletes != 0 {
			t.Error("deleted usable route after one lost reply")
		}
		mu.Unlock()
		time.Sleep(time.Second)
		synctest.Wait()
		u = <-updates
		if u.State != "active" || u.Address != "203.0.113.2:5555" {
			t.Fatal("new endpoint was not published", u)
		}
		cancel()
		<-done
		mu.Lock()
		defer mu.Unlock()
		if deletes != 1 {
			t.Fatalf("cleanup count %d", deletes)
		}
	})
}
func TestExpiredLeaseTriggersRediscoveryAfterRouterRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		discoveries := 0
		deletes := 0
		discover := func(context.Context) (*Mapping, error) {
			discoveries++
			attempt := discoveries
			return &Mapping{Address: "203.0.113.1:4747", Protocol: "fake", duration: 4 * time.Second,
				renew: func(_ context.Context, life uint32) (string, time.Duration, error) {
					if life == 0 {
						deletes++
						return "", 0, nil
					}
					if attempt == 1 {
						return "", 0, errors.New("router reboot")
					}
					return "203.0.113.3:4747", 120 * time.Second, nil
				}}, nil
		}
		var states []string
		done := make(chan struct{})
		go func() {
			defer close(done)
			maintain(ctx, discover, func(u Update) { states = append(states, u.State) })
		}()
		time.Sleep(12 * time.Second)
		synctest.Wait()
		cancel()
		<-done
		if discoveries < 2 {
			t.Fatal("did not rediscover router")
		}
		expired := false
		for _, s := range states {
			if s == "unavailable" {
				expired = true
			}
		}
		if !expired || deletes != discoveries {
			t.Fatalf("states %v cleanup %d/%d", states, deletes, discoveries)
		}
	})
}
func TestCancellationDuringDiscoveryReleasesLateLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		deleted := false
		entered := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			maintain(ctx, func(ctx context.Context) (*Mapping, error) {
				close(entered)
				<-ctx.Done()
				return &Mapping{duration: time.Minute, renew: func(context.Context, uint32) (string, time.Duration, error) { deleted = true; return "", 0, nil }}, nil
			}, func(Update) {})
		}()
		<-entered
		cancel()
		<-done
		if !deleted {
			t.Fatal("late lease leaked on cancellation")
		}
	})
}

func TestAlreadyExpiredLeaseIsNeverAdvertisedActive(t *testing.T) {
	deleted := false
	m := &Mapping{Address: "203.0.113.1:4747", duration: time.Minute, expires: time.Now().Add(-time.Second),
		renew: func(_ context.Context, life uint32) (string, time.Duration, error) {
			if life != 0 {
				t.Error("attempted renewal of already expired lease")
			}
			deleted = true
			return "", 0, nil
		}}
	m.run(context.Background(), func(u Update) {
		if u.State != "unavailable" {
			t.Error("advertised stale route", u)
		}
	})
	if !deleted {
		t.Fatal("expired lease cleanup not attempted")
	}
}
