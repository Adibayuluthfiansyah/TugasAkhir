package websocket

import "log"

const hubNotInitialized = "[WebSocket] Hub belum diinisialisasi"

func BroadcastToUser(userID, eventType, message string) {
	if HubInstance == nil {
		log.Println("[WebSocket] Hub belum diinisialisasi")
		return
	}

	event := NotificationEvent{
		UserID:  userID,
		Type:    eventType,
		Message: message,
	}

	HubInstance.Emit(event)
}

func BroadcastToAll(eventType, message string) {
	if HubInstance == nil {
		log.Println(hubNotInitialized)
		return
	}

	for userID := range HubInstance.clients {
		event := NotificationEvent{
			UserID:  userID,
			Type:    eventType,
			Message: message,
		}
		HubInstance.Emit(event)
	}
}

func Broadcast(event NotificationEvent) {
	if HubInstance == nil {
		log.Println(hubNotInitialized)
		return
	}
	HubInstance.Emit(event)
}

func BroadcastUserChanged(eventType string, payload interface{}) {
	if HubInstance == nil {
		log.Println(hubNotInitialized)
		return
	}

	HubInstance.mu.Lock()
	defer HubInstance.mu.Unlock()

	for userID := range HubInstance.clients {
		HubInstance.broadcast <- NotificationEvent{
			UserID:  userID,
			Type:    eventType,
			Payload: payload,
		}
	}
}
