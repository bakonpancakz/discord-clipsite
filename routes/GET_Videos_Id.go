package routes

import (
	"database/sql"
	"net/http"
	"shareclip/env"

	"github.com/gin-gonic/gin"
)

// Search Database for a video with the provided ID
func GET_Videos_ID(c *gin.Context) {
	var (
		VideoID      string
		VideoCreated string
		VideoStatus  string
	)
	err := env.DB.
		QueryRow("SELECT id, created, status FROM videos WHERE id = $1", c.Param("id")).
		Scan(&VideoID, &VideoCreated, &VideoStatus)

	switch {
	case err == sql.ErrNoRows:
		c.AbortWithStatusJSON(http.StatusNotFound, "Unknown Video")
	case err != nil:
		c.AbortWithError(http.StatusInternalServerError, err)
	case VideoStatus != "FINISH":
		c.AbortWithStatusJSON(http.StatusNotFound, "Processing Video")
	default:
		c.JSON(http.StatusOK, gin.H{
			"id":      VideoID,
			"created": VideoCreated,
			"status":  VideoStatus,
		})
	}
}
