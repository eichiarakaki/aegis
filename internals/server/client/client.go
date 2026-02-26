package client

import (
	"encoding/json"
	"log"
	"net"

	"github.com/eichiarakaki/aegis/internals/health"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Target  string `json:"target"`
	Message string `json:"message"`
}

func handleHealthCheck(target string, conn net.Conn) {
	var response HealthResponse

	switch target {

	case "all":
		err := health.CheckAll()
		if err != nil {
			log.Fatal("Health check failed:", err)
		}

		response = HealthResponse{
			Status:  "OK",
			Target:  "all",
			Message: "All subsystems healthy",
		}

	case "data":
		err := health.DataHealthCheck()
		if err != nil {
			log.Fatal("Data health check failed:", err)
		}

		response = HealthResponse{
			Status:  "OK",
			Target:  "data",
			Message: "Data quality and availability healthy",
		}

	case "sessions":
		err := health.SessionsHealthCheck()
		if err != nil {
			log.Fatal("Session manager health check failed:", err)
		}

		response = HealthResponse{
			Status:  "OK",
			Target:  "sessions",
			Message: "Session manager healthy",
		}

	default:
		response = HealthResponse{
			Status:  "ERROR",
			Target:  target,
			Message: "Unknown health target",
		}
	}

	json.NewEncoder(conn).Encode(response)
}

func HandleAegis(conn net.Conn) {
	defer conn.Close()

	var cmd Command
	err := json.NewDecoder(conn).Decode(&cmd)
	if err != nil {
		log.Println("Invalid command:", err)
		return
	}

	log.Printf("Received command: %s | Payload: %s\n", cmd.Type, cmd.Payload)

	switch cmd.Type {

	case "SESSION_START":
		log.Println("Starting session:", cmd.Payload)

	case "SESSION_STOP":
		log.Println("Stopping session:", cmd.Payload)

	case "SESSION_LIST":
		log.Println("Listing sessions")

	case "COMPONENT_LIST":
		log.Println("Listing components for session:", cmd.Payload)

	case "COMPONENT_GET":
		log.Println("Getting component:", cmd.Payload)

	case "COMPONENT_DESCRIBE":
		log.Println("Describing component:", cmd.Payload)

	case "HEALTH_CHECK":
		handleHealthCheck(cmd.Payload, conn)

	default:
		log.Println("Unknown command:", cmd.Type)
	}
}
