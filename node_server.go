package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Run serves an initialized node until the HTTP server returns an error.
func (n *Node) Run(addr string) error {
	if n == nil || n.Blockchain == nil || n.Mempool == nil || len(n.Blockchain.Blocks) == 0 {
		return fmt.Errorf("node is not initialized")
	}
	if n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1] == nil || n.Blockchain.UTXOSet == nil {
		return fmt.Errorf("blockchain state is not initialized")
	}
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	handler := n.Handler()
	// net/http serves requests concurrently; the v1 core uses ordinary maps.
	// Serialize requests at this runtime boundary without changing core APIs.
	var requests sync.Mutex
	server := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Lock()
			defer requests.Unlock()
			handler.ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
}
