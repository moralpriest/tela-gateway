package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/moralpriest/tela-gateway"
)

func main() {
	ln, addr, err := listen()
	if err != nil {
		log.Fatal(err)
	}

	base := "http://" + addr
	fmt.Printf("tela-gateway local: %s/\n", base)
	fmt.Printf("  health:    %s/health\n", base)
	fmt.Printf("  derobeats: %s/durl/derobeats.tela\n", base)
	fmt.Printf("  explorer:  %s/durl/explorer.tela\n", base)
	fmt.Printf("  any scid:  %s/scid/<64-hex-index-scid>/\n", base)
	fmt.Printf("daemons: %s\n", envOr("DERO_DAEMON_URLS", "127.0.0.1:10102,"+gateway.BundledNodes()))

	log.Fatal(http.Serve(ln, http.HandlerFunc(gateway.ServeTELA)))
}

func listen() (net.Listener, string, error) {
	if p := os.Getenv("PORT"); p != "" {
		ln, err := net.Listen("tcp", "127.0.0.1:"+p)
		if err != nil {
			return nil, "", fmt.Errorf("PORT=%s busy or invalid: %w", p, err)
		}
		return ln, ln.Addr().String(), nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	return ln, ln.Addr().String(), nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
