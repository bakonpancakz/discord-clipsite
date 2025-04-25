package routes

import (
	"net/http"
	"shareclip/env"
	"shareclip/tools"

	"github.com/gin-gonic/gin"
)

// Fetch all videos for the currently logged in user
func GET_Videos(c *gin.Context) {
	userSession := c.MustGet("user").(tools.RequestUser)
	userVideos := []gin.H{}
	rows, err := env.DB.Query(
		"SELECT id, created, status FROM videos WHERE user_id = $1",
		userSession.ID,
	)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var (
			VideoID      string
			VideoCreated string
			VideoStatus  string
		)
		if err := rows.Scan(&VideoID, &VideoCreated, &VideoStatus); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		userVideos = append(userVideos, gin.H{
			"id":      VideoID,
			"created": VideoCreated,
			"status":  VideoStatus,
		})
	}
	c.JSON(http.StatusOK, userVideos)
}
