package env

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path"
	"sync"
	"time"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

var (
	//go:embed Schema.sql
	databaseSchema  string
	databaseStarted sync.Once
	DB              *sql.DB
)

func StartDatabase(stop context.Context, await *sync.WaitGroup) {
	databaseStarted.Do(func() {
		t := time.Now()
		p := path.Join(DATA_DIR, "database.db")

		// Initialize Database
		if _, err := os.Stat(p); os.IsNotExist(err) {
			log.Println("[env/db] Creating Database, Welcome!")

			// Create New File
			f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, FILE_MODE)
			if err != nil {
				log.Fatalln("[env/db] Create Database Error:", err)
			}
			f.Close()
		}

		// Open Database
		var err error
		DB, err = sql.Open("sqlite3", p)
		if err != nil {
			log.Fatalln("[env/db] Open Database Error:", err)
		}
		DB.SetMaxOpenConns(1)
		if err := DB.Ping(); err != nil {
			log.Fatalln("[env/db] Ping Database Error:", err)
		}

		// Apply Schema
		if _, err := DB.Exec(databaseSchema); err != nil {
			log.Fatalln("[env/db] Cannot Apply Schema:", err)
		}

		// Shutdown Logic
		await.Add(1)
		go func() {
			defer await.Done()
			<-stop.Done()
			DB.Close()
			log.Println("[env/db] Database Closed")
		}()

		log.Println("[env/db] Ready in", time.Since(t))
	})
}
