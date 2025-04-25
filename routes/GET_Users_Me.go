package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Return already retrieved information about the current session
func GET_Users_Me(c *gin.Context) {
	c.JSON(http.StatusOK, c.MustGet("user"))
}
