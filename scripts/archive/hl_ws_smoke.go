package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := os.Getenv("HL_WS_URL")
	if url == "" {
		url = "wss://api.hyperliquid.xyz/ws"
	}
	log.Printf("Dialing %s", url)
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()
	log.Println("connected")

	// subscribe to trades for BTC as a smoke test
	sub := []byte(`{"method":"subscribe","subscription":{"type":"trades","coin":"BTC"}}`)
	if err := conn.WriteMessage(websocket.TextMessage, sub); err != nil {
		log.Fatalf("write subscribe error: %v", err)
	}
	log.Println("subscribed: trades BTC")

	conn.SetReadDeadline(time.Now().Add(8 * time.Second))
	for i := 0; i < 3; i++ { // read a few frames
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read error: %v", err)
		}
		fmt.Printf("[%d] %s\n", mt, string(msg))
	}
	log.Println("ok")
}
