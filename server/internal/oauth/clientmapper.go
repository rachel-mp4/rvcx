package oauth

import (
	"context"
	"errors"
	"sync"
	"time"
)

type ClientMap struct {
	svc     *Service
	clients map[int]*OauthXRPCClient
	expiry  map[int]time.Time
	texp    map[int]time.Time
	mu      sync.Mutex
}

func NewClientMap(service *Service) *ClientMap {
	return &ClientMap{
		svc:     service,
		clients: make(map[int]*OauthXRPCClient, 10),
		expiry:  make(map[int]time.Time, 10),
		texp:    make(map[int]time.Time, 10),
		mu:      sync.Mutex{},
	}
}

func (c *ClientMap) Map(id int, ctx context.Context) (cli *OauthXRPCClient, refreshed bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cli = c.clients[id]
	if cli == nil {
		return
	}

	texp := c.texp[id]
	expiry := c.expiry[id]
	if time.Now().After(expiry) {
		c.Delete(id)
		err = errors.New("client has expired")
		return
	}
	if texp.Sub(time.Now()) <= 5*time.Minute {
		var newexp time.Time
		newexp, err = c.svc.RefreshToken(ctx, cli.session)
		if err != nil {
			err = errors.New("failed to refresh expired token: " + err.Error())
			return
		}
		refreshed = true
		c.texp[id] = newexp
	}
	return
}

func (c *ClientMap) Append(id int, client *OauthXRPCClient, expiration time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[id] = client
	c.expiry[id] = expiration
	c.texp[id] = time.Now()

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
			delete(c.texp, id)
			continue
		}
		if client == nil {
			delete(c.expiry, id)
			delete(c.clients, id)
			delete(c.texp, id)
			continue
		}
		if now.After(expiry) {
			delete(c.expiry, id)
			delete(c.clients, id)
			delete(c.texp, id)
			continue
		}
	}
}

func (c *ClientMap) Delete(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.clients, id)
	delete(c.expiry, id)
	delete(c.texp, id)
}
