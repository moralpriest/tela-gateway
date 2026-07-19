package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/moralpriest/tela-gateway"
)

// VPS / Lambda Web Adapter / container: listen on 0.0.0.0:$PORT (default 8080).
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tela-gateway on http://%s/\n", ln.Addr().String())
	fmt.Printf("  DERO_DAEMON_URLS=%s\n", envOr("DERO_DAEMON_URLS", "127.0.0.1:10102,"+gateway.BundledNodes()))
	log.Fatal(http.Serve(ln, http.HandlerFunc(gateway.ServeTELA)))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
