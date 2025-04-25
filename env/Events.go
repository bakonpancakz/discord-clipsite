package env

import (
	"encoding/json"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	EventChannels = map[string]chan string{}
	EventMutex    sync.RWMutex
)

// Send a Event to the User via SSE (if connected)
// - eventType: The Event Type capitilized and written using the snake case naming convention (e.g. MY_EVENT)
// - eventSubject: The Relevant User or Video ID
// - data: Relevant Data, if any.
func SendEvent(userID string, eventType string, eventSubject string, data any) error {
	return SendMessage(userID, gin.H{
		"t": eventType,
		"s": eventSubject,
		"d": data,
	})
}

// Send a Message to the User via SSE (if connected)
func SendMessage(userID string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	EventMutex.RLock()
	if ch, ok := EventChannels[userID]; ok {
		select {
		case ch <- string(b):
		default:
		}
	}
	EventMutex.RUnlock()
	return nil
}
