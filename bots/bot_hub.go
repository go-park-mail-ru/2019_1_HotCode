package bots

var h *hub

type hub struct {
	// UserID -> GameID -> SessionID -> byte channel
	sessions map[int64]map[string]map[string]chan *BotVerifyStatusMessage

	broadcast  chan *BotVerifyStatusMessage
	register   chan *BotVerifyClient
	unregister chan *BotVerifyClient
}

func (h *hub) registerClient(client *BotVerifyClient) {
	if _, ok := h.sessions[client.UserID]; !ok {
		h.sessions[client.UserID] = make(map[string]map[string]chan *BotVerifyStatusMessage)
	}

	if _, ok := h.sessions[client.UserID][client.GameSlug]; !ok {
		h.sessions[client.UserID][client.GameSlug] = make(map[string]chan *BotVerifyStatusMessage)
	}

	h.sessions[client.UserID][client.GameSlug][client.SessionID] = client.send
}

func (h *hub) unregisterClient(client *BotVerifyClient) {
	if _, ok := h.sessions[client.UserID]; ok {
		if _, ok := h.sessions[client.UserID][client.GameSlug]; ok {
			if _, ok := h.sessions[client.UserID][client.GameSlug][client.SessionID]; ok {
				delete(h.sessions[client.UserID][client.GameSlug], client.SessionID)
				close(client.send)
			}

			if len(h.sessions[client.UserID][client.GameSlug]) == 0 {
				delete(h.sessions[client.UserID], client.GameSlug)
			}
		}

		if len(h.sessions[client.UserID]) == 0 {
			delete(h.sessions, client.UserID)
		}
	}
}

func (h *hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			// для тех, кто слушает какую-то одну игру
			for _, send := range h.sessions[message.AuthorID][message.GameSlug] {
				send <- message
			}

			// для тех, кто слушает профиль игрока
			for _, send := range h.sessions[message.AuthorID][""] {
				send <- message
			}
		}
	}
}

func init() {
	h = &hub{
		sessions:   make(map[int64]map[string]map[string]chan *BotVerifyStatusMessage),
		broadcast:  make(chan *BotVerifyStatusMessage),
		register:   make(chan *BotVerifyClient),
		unregister: make(chan *BotVerifyClient),
	}

	go h.run()
}
