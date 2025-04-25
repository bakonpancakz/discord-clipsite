package env

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

const (
	FILE_MODE       = os.FileMode(0600) // Read/Write for the Current User
	MAX_FILE_SIZE   = 4 << 30           // Limited to 4 GB
	COOKIE_LIFETIME = 7 * 24 * 60 * 60  // 7 days
)

var (
	HTTP_TLS          *tls.Config                                   // http: TLS Configuration
	DATA_DIR          = EnvString("DATA", "data")                   // Data Directory
	HTTP_BIND         = EnvString("HTTP_BIND", "localhost:8080")    // http: Address to Listen for Requests on
	TLS_ENABLED       = EnvString("TLS_ENABLED", "false") == "true" // http: Enable TLS?
	TLS_CERT          = EnvString("TLS_CERT", "tls_crt.pem")        // http: Path to TLS Certificate
	TLS_KEY           = EnvString("TLS_KEY", "tls_key.pem")         // http: Path to TLS Key
	TLS_CA            = EnvString("TLS_CA", "tls_ca.pem")           // http: Path to TLS CA Bundle
	DISCORD_REDIRECT  = EnvString("DISCORD_REDIRECT", "")           // Discord: Application Redirect URI
	DISCORD_CLIENT_ID = EnvString("DISCORD_CLIENT_ID", "")          // Discord: Application Client ID
	DISCORD_SECRET    = EnvString("DISCORD_SECRET", "")             // Discord: Application Secret Key
)

func init() {
	// Initialize Data Directories
	for _, dirname := range []string{"public", "video"} {
		if err := os.MkdirAll(path.Join(DATA_DIR, dirname), FILE_MODE); err != nil {
			log.Fatalln("[env/data]", err)
		}
	}

	// Load and Parse TLS Configuration from Disk
	if TLS_ENABLED {
		cert, err := tls.LoadX509KeyPair(TLS_CERT, TLS_KEY)
		if err != nil {
			log.Fatalln("[env/tls] Cannot Load Keypair", err)
		}
		caBytes, err := os.ReadFile(TLS_CA)
		if err != nil {
			log.Fatalln("[env/tls] Cannot Read CA File", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caBytes) {
			log.Fatalln("[env/tls] Cannot Append Certificates")
		}
		HTTP_TLS = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    caPool,
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		}
	}
}

// Reads String from Environment
func EnvString(key, defaultValue string) string {
	systemValue := os.Getenv(key)
	if systemValue == "" {
		if defaultValue == "" {
			fmt.Printf("Variable '%s' was not set\n", key)
			os.Exit(2)
		}
		return defaultValue
	}
	return systemValue
}

// Read Number from Environment
func EnvNumber(key string, defaultValue int) int {
	systemValue := os.Getenv(key)
	if systemValue == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(systemValue)
	if err != nil {
		fmt.Printf("Variable '%s' is not a integer: %s\n", key, err)
		os.Exit(2)
	}
	return v
}
