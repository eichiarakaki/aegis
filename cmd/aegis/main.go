package main

import (
	"encoding/json"
	"log"
	"net"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var cmd Command
	err := json.NewDecoder(conn).Decode(&cmd)
	if err != nil {
		log.Println("Invalid command:", err)
		return
	}

	log.Printf("Received command: %s | Payload: %s\n", cmd.Type, cmd.Payload)

	switch cmd.Type {
	case "START_SESSION":
		log.Println("Starting session...")
	case "STOP_SESSION":
		log.Println("Stopping session...")
	default:
		log.Println("Unknown command")
	}
}

func main() {
	listener, err := net.Listen("tcp", ":7000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("Aegis daemon listening on :7000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}
