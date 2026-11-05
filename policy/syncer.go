package policy

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/redis/go-redis/v9"
)

// Watcher implements a lightweight distributed policy synchronization watcher backed by Redis Pub/Sub.
type Watcher struct {
	client     *redis.Client
	channel    string
	callback   func(string)
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	closed     bool
}

// NewRedisWatcher initializes a new Redis watcher for Casbin policy updates.
func NewRedisWatcher(redisURL, channel string) (*Watcher, error) {
	if channel == "" {
		channel = "casbin_policy_update"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback to direct address if simple host:port is provided
		opt = &redis.Options{
			Addr: redisURL,
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Best-effort ping check
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[Syncer] Redis connection warning: %v (running with local memory sync)", err)
	}

	wCtx, wCancel := context.WithCancel(context.Background())
	w := &Watcher{
		client:     client,
		channel:    channel,
		cancelFunc: wCancel,
	}

	go w.listen(wCtx)
	return w, nil
}

// SetUpdateCallback sets the callback function executed when a policy change event is received.
func (w *Watcher) SetUpdateCallback(callback func(string)) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callback = callback
	return nil
}

// Update broadcasts a policy mutation message across all active instances.
func (w *Watcher) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed || w.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	msg := fmt.Sprintf("policy_updated:%d", time.Now().UnixNano())
	return w.client.Publish(ctx, w.channel, msg).Err()
}

// Close gracefully terminates the watcher and Redis connection.
func (w *Watcher) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}
	w.closed = true
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	if w.client != nil {
		_ = w.client.Close()
	}
}

func (w *Watcher) listen(ctx context.Context) {
	pubsub := w.client.Subscribe(ctx, w.channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.mu.Lock()
			cb := w.callback
			w.mu.Unlock()

			if cb != nil {
				cb(msg.Payload)
			}
		}
	}
}

// PolicyManager coordinates the enforcer and distributed syncer.
type PolicyManager struct {
	Enforcer *casbin.Enforcer
	Watcher  *Watcher
	mu       sync.RWMutex
}

// NewPolicyManager creates a new thread-safe PolicyManager.
func NewPolicyManager(e *casbin.Enforcer, w *Watcher) *PolicyManager {
	pm := &PolicyManager{
		Enforcer: e,
		Watcher:  w,
	}

	if w != nil {
		_ = w.SetUpdateCallback(func(payload string) {
			pm.mu.Lock()
			defer pm.mu.Unlock()
			log.Printf("[Syncer] Received update notification: %s. Reloading policies...", payload)
			_ = pm.Enforcer.LoadPolicy()
		})
	}

	return pm
}

// AddNamedPolicy adds a policy rule and broadcasts the change.
func (pm *PolicyManager) AddNamedPolicy(ptype string, params ...string) (bool, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var added bool
	var err error
	if ptype == "g" || ptype == "g2" || ptype == "g3" {
		added, err = pm.Enforcer.AddNamedGroupingPolicy(ptype, params)
	} else {
		added, err = pm.Enforcer.AddNamedPolicy(ptype, params)
	}
	if err != nil || !added {
		return added, err
	}

	if pm.Watcher != nil {
		_ = pm.Watcher.Update()
	}
	return true, nil
}

// Enforce evaluates permissions thread-safely.
func (pm *PolicyManager) Enforce(rvals ...interface{}) (bool, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.Enforcer.Enforce(rvals...)
}
