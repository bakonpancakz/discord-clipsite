package routes

import (
	"io"
	"shareclip/env"
	"shareclip/tools"

	"github.com/gin-gonic/gin"
)

// Serve Events to the Relevant User ID
func GET_Events(c *gin.Context) {
	userSession := c.MustGet("user").(tools.RequestUser)

	// Create Events for User
	EVENTS := make(chan string, 16)
	env.EventMutex.Lock()
	env.EventChannels[userSession.ID] = EVENTS
	env.EventMutex.Unlock()
	defer func() {
		env.EventMutex.Lock()
		delete(env.EventChannels, userSession.ID)
		close(EVENTS)
		env.EventMutex.Unlock()
	}()
	env.SendEvent(userSession.ID, "WELCOME", "", "")

	// Collect Events for User
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case e := <-EVENTS:
			c.SSEvent("data", e)
			return true
		}
	})
}
