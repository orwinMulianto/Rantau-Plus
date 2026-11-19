package assistant

import (
	"slices"
	"sync"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type conversation struct {
	messages []Message
	updated  time.Time
	timer    *time.Timer
}

// Conversation context is isolated by authenticated session, bounded, and kept in RAM.
type conversations struct {
	mu       sync.Mutex
	sessions map[string]*conversation
}

func (c *conversations) read(session string) []Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry := c.sessions[session]; entry != nil && time.Since(entry.updated) < 30*time.Minute {
		return slices.Clone(entry.messages)
	}
	return nil
}

func (c *conversations) remember(session, question, reply string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessions == nil {
		c.sessions = make(map[string]*conversation)
	}
	entry := c.sessions[session]
	if entry == nil && len(c.sessions) >= 512 {
		var oldest string
		for key, value := range c.sessions {
			if oldest == "" || value.updated.Before(c.sessions[oldest].updated) {
				oldest = key
			}
		}
		c.sessions[oldest].timer.Stop()
		delete(c.sessions, oldest)
	}
	var previous []Message
	if entry != nil {
		entry.timer.Stop()
		if time.Since(entry.updated) < 30*time.Minute {
			previous = entry.messages
		}
	}
	runes := []rune(reply)
	entry = &conversation{updated: time.Now(), messages: append(previous, Message{"user", question}, Message{"assistant", string(runes[:min(len(runes), 1500)])})}
	entry.messages = slices.Clone(entry.messages[max(0, len(entry.messages)-8):])
	c.sessions[session] = entry
	entry.timer = time.AfterFunc(30*time.Minute, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.sessions[session] == entry {
			delete(c.sessions, session)
		}
	})
}
