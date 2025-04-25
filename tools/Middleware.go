package tools

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shareclip/env"
	"time"

	"github.com/gin-gonic/gin"
)

// A User available via the Request Keys
type RequestUser struct {
	ID     string  `json:"id"`     // Their Discord ID
	Avatar *string `json:"avatar"` // Their Discord Avatar Hash
	Name   string  `json:"name"`   // Their Discord Username/Displayname
}

// Lookup the User via their Session Cookie
func Session(c *gin.Context) {

	// Parse Sent Cookies
	token, err := c.Cookie("session")
	if err != nil {
		if err == http.ErrNoCookie {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Login Required")
			return
		}
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, "Invalid or Malformed Cookie(s)")
		return
	}

	// Lookup User via Cookie
	var user RequestUser
	err = env.DB.
		QueryRow("SELECT id, avatar, name FROM users WHERE token = $1", token).
		Scan(&user.ID, &user.Avatar, &user.Name)

	switch {
	case err == sql.ErrNoRows:
		c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthorized")
	case err != nil:
		c.AbortWithError(http.StatusInternalServerError, err)
	default:
		c.Set("user", user)
	}
}

// Logs Requests to the Application Log
func Logger(c *gin.Context) {
	var RequestStart = time.Now()
	var RequestError = "[]"
	c.Next()

	// Convert Errors into JSON String Array
	if len(c.Errors) > 0 {
		Messages := make([]string, len(c.Errors))
		for i := range c.Errors {
			Messages[i] = c.Errors[i].Error()
		}
		if b, err := json.Marshal(Messages); err != nil {
			fmt.Println("Unable to Marshal Error Messages:", err)
		} else {
			RequestError = string(b)
		}
	}

	// Retrieve User ID (if logged in)
	var UserID string
	if v, ok := c.Get("user"); ok {
		if u, ok := v.(RequestUser); ok {
			UserID = u.ID
		}
	}

	// Log Request to Console
	log.Printf(
		"[http] ip=%s uid=%s s=%d m=%s p=\"%s\" bi=%d bo=%d l=%s err=\"%s\" ref=\"%s\" dev=\"%s\"\n",
		c.ClientIP(),
		UserID,
		c.Writer.Status(),
		c.Request.Method,
		c.Request.RequestURI,
		c.Request.ContentLength,
		c.Writer.Size(),
		time.Since(RequestStart),
		RequestError,
		c.Request.Referer(),
		c.Request.UserAgent(),
	)
}
