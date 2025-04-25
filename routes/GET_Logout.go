package routes

import (
	"net/http"
	"shareclip/env"
	"shareclip/tools"

	"github.com/gin-gonic/gin"
)

// Delete Token for Current User in Database
func GET_Logout(c *gin.Context) {
	userSession := c.MustGet("user").(tools.RequestUser)
	_, err := env.DB.Exec(
		"UPDATE users SET token = NULL WHERE id = $1",
		userSession.ID,
	)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.SetCookie("session", "", -1, "", "", env.TLS_ENABLED, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
