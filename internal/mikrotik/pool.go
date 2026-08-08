package mikrotik

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	ros "github.com/go-routeros/routeros/v3"
)

// Pool manages reusable MikroTik API connections per router address.
// Idle connections are cleaned up after 5 minutes of inactivity.
type Pool struct {
	mu    sync.Mutex
	conns map[string]*poolEntry
}

type poolEntry struct {
	client     *ros.Client
	lastUsedAt time.Time
	addr       string
	username   string
	password   string
}

var (
	instance *Pool
	once     sync.Once
)

// GetPool returns the singleton Pool instance.
func GetPool() *Pool {
	once.Do(func() {
		instance = &Pool{
			conns: make(map[string]*poolEntry),
		}
		go instance.cleanupLoop()
	})
	return instance
}

// GetClient returns a reusable MikroTik client for the given address.
// If an existing connection is alive, it is reused; otherwise a new one is created.
func (p *Pool) GetClient(addr, username, password string, timeout time.Duration) (*ros.Client, error) {
	key := fmt.Sprintf("%s|%s", addr, username)

	p.mu.Lock()
	if entry, ok := p.conns[key]; ok {
		// Check if the connection is still usable
		if entry.client != nil {
			entry.lastUsedAt = time.Now()
			client := entry.client
			p.mu.Unlock()
			return client, nil
		}
		// Dead connection, remove it
		delete(p.conns, key)
	}
	p.mu.Unlock()

	// Create new connection
	client, err := dialMikrotik(addr, username, password, timeout)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.conns[key] = &poolEntry{
		client:     client,
		lastUsedAt: time.Now(),
		addr:       addr,
		username:   username,
		password:   password,
	}
	p.mu.Unlock()

	return client, nil
}

// Close closes a specific connection (e.g., after an error).
func (p *Pool) Close(addr, username string) {
	key := fmt.Sprintf("%s|%s", addr, username)
	p.mu.Lock()
	if entry, ok := p.conns[key]; ok {
		if entry.client != nil {
			entry.client.Close()
		}
		delete(p.conns, key)
	}
	p.mu.Unlock()
}

// CloseAll closes all pooled connections (used during graceful shutdown).
func (p *Pool) CloseAll() {
	p.mu.Lock()
	for _, entry := range p.conns {
		if entry.client != nil {
			entry.client.Close()
		}
	}
	p.conns = make(map[string]*poolEntry)
	p.mu.Unlock()
}

// cleanupLoop runs periodically to close idle connections older than 5 minutes.
func (p *Pool) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		p.cleanupIdle(5 * time.Minute)
	}
}

func (p *Pool) cleanupIdle(maxIdle time.Duration) {
	now := time.Now()
	p.mu.Lock()
	for key, entry := range p.conns {
		if now.Sub(entry.lastUsedAt) > maxIdle {
			if entry.client != nil {
				entry.client.Close()
			}
			delete(p.conns, key)
		}
	}
	p.mu.Unlock()
}

// Stats returns basic pool statistics for monitoring.
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]interface{}{
		"totalConnections": len(p.conns),
		"maxIdleSeconds":   300,
	}
}

// dialMikrotik connects to a MikroTik router API, automatically using TLS for
// port 8729 (SSL API) and plain TCP for port 8728 (non-SSL API).
func dialMikrotik(addr, username, password string, timeout time.Duration) (*ros.Client, error) {
	if strings.HasSuffix(addr, ":8729") {
		tlsConfig := &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		return ros.DialTLSTimeout(addr, username, password, tlsConfig, timeout)
	}
	return ros.DialTimeout(addr, username, password, timeout)
}
