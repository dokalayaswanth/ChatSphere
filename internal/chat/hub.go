package chat

type broadcastEvent struct {
	RoomID  int64
	Payload []byte
}

type Hub struct {
	register   chan *Client
	unregister chan *Client
	join       chan roomMembershipEvent
	leave      chan roomMembershipEvent
	broadcast  chan broadcastEvent
	rooms      map[int64]map[*Client]struct{}
}

type roomMembershipEvent struct {
	Client *Client
	RoomID int64
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		join:       make(chan roomMembershipEvent),
		leave:      make(chan roomMembershipEvent),
		broadcast:  make(chan broadcastEvent, 256),
		rooms:      make(map[int64]map[*Client]struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if client.joinedRooms == nil {
				client.joinedRooms = make(map[int64]struct{})
			}
		case client := <-h.unregister:
			for roomID := range client.joinedRooms {
				delete(h.rooms[roomID], client)
			}
			close(client.send)
		case event := <-h.join:
			roomClients, ok := h.rooms[event.RoomID]
			if !ok {
				roomClients = make(map[*Client]struct{})
				h.rooms[event.RoomID] = roomClients
			}
			roomClients[event.Client] = struct{}{}
			event.Client.joinedRooms[event.RoomID] = struct{}{}
		case event := <-h.leave:
			delete(h.rooms[event.RoomID], event.Client)
			delete(event.Client.joinedRooms, event.RoomID)
		case event := <-h.broadcast:
			for client := range h.rooms[event.RoomID] {
				select {
				case client.send <- event.Payload:
				default:
					delete(h.rooms[event.RoomID], client)
					delete(client.joinedRooms, event.RoomID)
					close(client.send)
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Join(client *Client, roomID int64) {
	h.join <- roomMembershipEvent{Client: client, RoomID: roomID}
}

func (h *Hub) Leave(client *Client, roomID int64) {
	h.leave <- roomMembershipEvent{Client: client, RoomID: roomID}
}

func (h *Hub) Broadcast(roomID int64, payload []byte) {
	h.broadcast <- broadcastEvent{RoomID: roomID, Payload: payload}
}
