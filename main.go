package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"shareclip/env"
	"shareclip/routes"
	"shareclip/tools"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Startup Services
	var stopCtx, stop = context.WithCancel(context.Background())
	var stopWg sync.WaitGroup
	env.StartDatabase(stopCtx, &stopWg)
	env.StartEncoders(stopCtx, &stopWg)
	routes.SetupSPA()
	SetupHTTP(stopCtx, &stopWg)

	// Await Shutdown Signal
	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGINT, syscall.SIGTERM)
	<-cancel
	stop()

	// Begin Shutdown Process
	timeout, finish := context.WithTimeout(context.Background(), time.Minute)
	defer finish()
	go func() {
		<-timeout.Done()
		if timeout.Err() == context.DeadlineExceeded {
			log.Fatalln("[main] Cleanup timeout! Exiting now.")
		}
	}()
	stopWg.Wait()
	log.Println("[main] All done, bye bye!")
	os.Exit(1)
}

func SetupHTTP(stop context.Context, await *sync.WaitGroup) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(tools.Logger)
	r.Static("/public", path.Join(env.DATA_DIR, "public"))
	r.GET("/api/oauth2", routes.GET_oAuth2_Callback)
	r.GET("/api/logout", tools.Session, routes.GET_Logout)
	r.GET("/api/events", tools.Session, routes.GET_Events)
	r.POST("/api/videos", tools.Session, routes.POST_Upload)
	r.GET("/api/videos", tools.Session, routes.GET_Videos)
	r.GET("/api/videos/:id", routes.GET_Videos_ID)
	r.GET("/api/users/@me", tools.Session, routes.GET_Users_Me)
	r.GET("/robots.txt", func(c *gin.Context) {
		c.String(http.StatusOK, "User-agent: *\nDisallow: /")
	})
	r.NoRoute(routes.GET_Index)

	svr := http.Server{
		Handler:           r,
		Addr:              env.HTTP_BIND,
		TLSConfig:         env.HTTP_TLS,
		MaxHeaderBytes:    4096,
		IdleTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		// causes bad side effects
		// WriteTimeout:      60 * time.Second,
		// ReadTimeout:       60 * time.Second,
	}

	// Shutdown Logic
	await.Add(1)
	go func() {
		defer await.Done()
		<-stop.Done()
		svr.Shutdown(context.Background())
		log.Println("[http] Cleaned up HTTP")
	}()

	// Server Startup
	var err error
	log.Printf("[http] Setup HTTP (Addr: %s)\n", svr.Addr)
	if env.TLS_ENABLED {
		err = svr.ListenAndServeTLS("", "")
	} else {
		err = svr.ListenAndServe()
	}
	if err != http.ErrServerClosed {
		log.Fatalln("[http] Listen Error:", err)
	}
}
