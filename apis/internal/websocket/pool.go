package websocket

import "github.com/rs/zerolog/log"

const (
	UserConnectionType = 2
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Broadcast  chan Message
	History    []Message
	Stopped    chan struct{}
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		History:    make([]Message, 0),
		Stopped:    make(chan struct{}),
	}
}

func (pool *Pool) handleRegister(client *Client) {
	pool.Clients[client] = true

	for _, message := range pool.History {
		if err := client.Conn.WriteJSON(message); err != nil {
			log.Logger.Error().Err(err).Msg("failed to write jsonn")
		}
	}
	joinMessage := Message{Type: UserConnectionType, Body: "New user joined"}
	pool.History = append(pool.History, joinMessage)
	for client := range pool.Clients {
		if err := client.Conn.WriteJSON(joinMessage); err != nil {
			log.Logger.Error().Err(err).Msg("failed to write jsonn")
		}
	}
}

func (pool *Pool) handleUnregister(client *Client) {
	delete(pool.Clients, client)

	disconnectMessage := Message{Type: UserConnectionType, Body: "User disconnected"}
	pool.History = append(pool.History, disconnectMessage)
	for client := range pool.Clients {
		if err := client.Conn.WriteJSON(disconnectMessage); err != nil {
			log.Logger.Error().Err(err).Msg("failed to write jsonn")
		}
	}
}

func (pool *Pool) handleBroadcast(message Message) {
	pool.History = append(pool.History, message)
	for client := range pool.Clients {
		if err := client.Conn.WriteJSON(message); err != nil {
			log.Logger.Error().Err(err).Msg("failed to write jsonn")
		}
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			pool.handleRegister(client)
		case client := <-pool.Unregister:
			pool.handleUnregister(client)
		case message := <-pool.Broadcast:
			pool.handleBroadcast(message)
		case <-pool.Stopped:
			return
		}
	}
}

func (pool *Pool) Stop() {
	close(pool.Stopped)
}
