package routes

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"shareclip/env"
	"shareclip/tools"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var IndexOriginal []byte   // Modified Version of Site
var IndexCompressed []byte // Modified and Compressed Version of Site
var IndexETag string       // MD5 Hash of Site

func SetupSPA() {
	// Replace Template Strings
	IndexOriginal = bytes.ReplaceAll(IndexOriginal, []byte("{{filename_video}}"), []byte(env.OUTPUT_FILENAME_VIDEO))
	IndexOriginal = bytes.ReplaceAll(IndexOriginal, []byte("{{filename_thumb}}"), []byte(env.OUTPUT_FILENAME_THUMBNAIL))

	// Compress Webpage
	b := bytes.Buffer{}
	c, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
	c.Write(IndexOriginal)
	c.Flush()
	IndexCompressed = b.Bytes()

	// Generate Website ETag
	IndexETag = fmt.Sprintf("%X", md5.Sum(IndexOriginal))
	log.Printf(
		"[routes] Compressed Index.html (%.2fkb => %.2fkb)\n",
		float32(len(IndexOriginal))/1024,
		float32(len(IndexCompressed))/1024,
	)
}

func GET_Index(c *gin.Context) {

	// Ignore Unwanted Methods
	if c.Request.Method != http.MethodGet {
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	// Serve Video Embed for Discord
	if strings.Contains(c.GetHeader("User-Agent"), "Discord") {

		// Search Database for Video
		VideoID := strings.Trim(c.Request.URL.Path, "/")
		if !tools.IDMatcher.MatchString(VideoID) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		err := env.DB.
			QueryRow("SELECT id FROM videos WHERE id = $1 AND status = 'FINISH'", VideoID).
			Scan(&VideoID)

		// Render Embed Webpage
		switch {
		case err == sql.ErrNoRows:
			c.AbortWithStatus(http.StatusNotFound)
		case err != nil:
			c.AbortWithError(http.StatusInternalServerError, err)
		default:
			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, ""+
				"<!DOCTYPE html>"+
				"<html>"+
				/**/ "<head>"+
				/**/ /**/ "<title>Clips</title>"+
				/**/ /**/ "<meta property=\"og:type\" content=\"video.other\">"+
				/**/ /**/ "<meta property=\"og:image\" content=\"https://%[1]s/public/%[2]s/%[3]s\">"+
				/**/ /**/ "<meta property=\"og:video:url\" content=\"https://%[1]s/public/%[2]s/%[4]s\">"+
				/**/ /**/ "<meta property=\"og:video:width\" content=\"1920\">"+
				/**/ /**/ "<meta property=\"og:video:height\" content=\"1080\">"+
				/**/ "</head>"+
				"</html>",
				c.Request.Host,
				VideoID,
				env.OUTPUT_FILENAME_THUMBNAIL,
				env.OUTPUT_FILENAME_VIDEO,
			)
		}
		return
	}

	// Serve Single-Page Application

	// Cache Validation
	if c.GetHeader("If-None-Match") == IndexETag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("ETag", IndexETag)
	c.Header("Content-Type", "text/html")

	// Serve Compressed Version
	if strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		c.Header("Content-Encoding", "gzip")
		c.Data(http.StatusOK, "text/html", IndexCompressed)
		return
	}

	// Serve Standard Version
	c.Data(http.StatusOK, "text/html", IndexOriginal)
}
