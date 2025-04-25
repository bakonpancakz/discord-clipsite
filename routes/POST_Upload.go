package routes

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"shareclip/env"
	"shareclip/tools"

	"github.com/gin-gonic/gin"
)

var allowedExtensions = map[string]bool{
	".webm": true, // cutting edge
	".mp4":  true, // legacy
	".mov":  true, // iphone
}

// Upload and Queue a video for processing
func POST_Upload(c *gin.Context) {

	// Impose Body Size Limitations
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, env.MAX_FILE_SIZE)
	if c.Request.ContentLength > env.MAX_FILE_SIZE {
		c.AbortWithStatusJSON(http.StatusBadRequest, "Payload Too Large")
		return
	}
	formBoundary := ""
	formFileCount := 0
	if _, params, err := mime.ParseMediaType(c.GetHeader("Content-Type")); err != nil || params["boundary"] == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, "Invalid Content-Type")
		return
	} else {
		formBoundary = params["boundary"]
	}

	// Initialize Upload Directory
	var (
		userSession    = c.MustGet("user").(tools.RequestUser)
		uploadID       = tools.GenerateVideoID()
		uploadPath     = path.Join(env.DATA_DIR, "video", uploadID)
		uploadComplete = false
		uploadForm     = multipart.NewReader(c.Request.Body, formBoundary)
	)
	defer func() {
		if !uploadComplete {
			os.RemoveAll(uploadPath)
		}
	}()

	// Parse Form Fields
	var errorServer error
	var errorClient string
	for {
		formPart, err := uploadForm.NextPart()
		if err == io.EOF {
			// Placement is a little strange but we have to at minimum discard the incoming
			// form otherwise the browser will throw a socket error instead of our error message
			if errorServer != nil {
				c.AbortWithError(http.StatusInternalServerError, errorServer)
				return
			}
			if errorClient != "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, errorClient)
				return
			}
			break
		}
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		switch {
		case errorClient != "":
		case errorServer != nil:

		case formPart.FormName() == "video":
			if formPart.FileName() == "" {
				errorClient = "Expected Video File"
				continue
			}
			if formFileCount > 0 {
				errorClient = "Too Many Files"
				continue
			}
			if _, ok := allowedExtensions[path.Ext(formPart.FileName())]; !ok {
				errorClient = "Invalid File Type"
				continue
			}
			formFileCount++

			// Copy File to Disk
			f, err := os.OpenFile(uploadPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, env.FILE_MODE)
			if err != nil {
				errorServer = err
				continue
			}
			defer f.Close()
			if _, err := io.Copy(f, formPart); err != nil {
				errorServer = err
				continue
			}

		default:
			errorClient = "Invalid Form Body"
		}
	}
	if formFileCount == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, "No Video Uploaded")
		return
	}

	// Queue Video for Encoding
	_, err := env.DB.Exec(
		"INSERT INTO videos (id, user_id, status) VALUES ($1, $2, 'QUEUE')",
		uploadID, userSession.ID,
	)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	uploadComplete = true
	env.WakeEncoder()

	// Return Video ID
	c.JSON(http.StatusCreated, uploadID)
}
