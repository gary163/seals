package server

import (
	"sync"
)

type KEY interface{}

type channel struct {
	mu sync.RWMutex
	sessions map[KEY]*Session
}

func NewChannel() *channel{
	channel := &channel{}
	channel.sessions = make(map[KEY]*Session)
	return channel
}

func (c *channel) Set(key KEY,session *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if session,ok := c.sessions[key]; ok {
		c.deleteLocked(key,session)
	}

	c.sessions[key] = session
	session.AddCloseCallback(c,key,func(){
		c.Delete(key)
	})
}

func (c *channel) Get(key KEY) *Session{
	c.mu.Lock()
	defer c.mu.Unlock()
	if session,ok := c.sessions[key]; ok {
		return session
	}
	return nil
}

func (c *channel) Delete(key KEY) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if session,ok := c.sessions[key]; ok {
		c.deleteLocked(key,session)
		return true
	}
	return false
}

func (c *channel) deleteLocked(key KEY, session *Session) {
	session.DelCloseCallback(c,key)
	delete(c.sessions,key)
}

func (c *channel) Fetch(callback func(session *Session)){
	c.mu.Lock()
	defer  c.mu.Unlock()

	for _,session := range c.sessions {
		if session != nil {
			callback(session)
		}
	}
}

func (c *channel) Destroy(){
	c.mu.Lock()
	defer c.mu.Unlock()
	for key,session := range c.sessions {
		c.deleteLocked(key,session)
	}
}