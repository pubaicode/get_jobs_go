package sse

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

type Event struct {
	Event string
	Data  string
}

type Broker struct {
	clients    map[chan Event]struct{}
	register   chan chan Event
	unregister chan chan Event
	broadcast  chan Event
	stop       chan struct{}
}

func NewBroker() *Broker {
	b := &Broker{
		clients:    make(map[chan Event]struct{}),
		register:   make(chan chan Event),
		unregister: make(chan chan Event),
		broadcast:  make(chan Event, 256),
		stop:       make(chan struct{}),
	}
	go b.run()
	return b
}

func (b *Broker) run() {
	for {
		select {
		case client := <-b.register:
			b.clients[client] = struct{}{}
		case client := <-b.unregister:
			if _, ok := b.clients[client]; ok {
				close(client)
				delete(b.clients, client)
			}
		case event := <-b.broadcast:
			for client := range b.clients {
				select {
				case client <- event:
				default:
					close(client)
					delete(b.clients, client)
				}
			}
		case <-b.stop:
			return
		}
	}
}

func (b *Broker) Subscribe() chan Event {
	ch := make(chan Event, 64)
	b.register <- ch
	return ch
}

func (b *Broker) Unsubscribe(ch chan Event) {
	b.unregister <- ch
}

func (b *Broker) Publish(event string, data string) {
	b.broadcast <- Event{Event: event, Data: data}
}

func (b *Broker) Stop() {
	close(b.stop)
}

func (b *Broker) ServeHTTP(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			if event.Event != "" {
				fmt.Fprintf(w, "event: %s\n", event.Event)
			}
			fmt.Fprintf(w, "data: %s\n\n", event.Data)
			return true
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": ping\n\n")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

type SSEManager struct {
	platforms map[string]*Broker
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		platforms: make(map[string]*Broker),
	}
}

func (m *SSEManager) GetBroker(platform string) *Broker {
	if b, ok := m.platforms[platform]; ok {
		return b
	}
	b := NewBroker()
	m.platforms[platform] = b
	return b
}

func (m *SSEManager) Publish(platform, event, data string) {
	if b, ok := m.platforms[platform]; ok {
		b.Publish(event, data)
	}
}
