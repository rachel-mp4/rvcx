package oauth

import (
	"sync"
	"time"
)

type ClientMap struct {
	clients map[uint]*OauthXRPCClient
	expiry  map[uint]time.Time
	mu      sync.Mutex
}

func NewClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[uint]*OauthXRPCClient, 10),
		expiry:  make(map[uint]time.Time, 10),
	}
}

func (c *ClientMap) Map(id uint) *OauthXRPCClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clients[id]
}

func (c *ClientMap) Append(id uint, client *OauthXRPCClient, expiration time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[id] = client
	c.expiry[id] = expiration
}

func (c *ClientMap) Cleanup() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, client := range c.clients {
		expiry, ok := c.expiry[id]
		if !ok {
			delete(c.expiry, id)
			delete(c.clients, id)
			continue
		}
		if client == nil {
			delete(c.expiry, id)
			delete(c.clients, id)
			continue
		}
		if expiry.After(now) {
			delete(c.expiry, id)
			delete(c.clients, id)
			continue
		}
	}
}
