package client

import (
	"encoding/json"
	"log"
	"net"

	"github.com/eichiarakaki/aegis/internals/health"
	"github.com/eichiarakaki/aegis/internals/logger"
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

	logger.Infof("Received command: %s | Payload: %s\n", cmd.Type, cmd.Payload)

	switch cmd.Type {

	case "SESSION_START":
		logger.Info("Starting session:", cmd.Payload)

	case "SESSION_STOP":
		logger.Info("Stopping session:", cmd.Payload)

	case "SESSION_LIST":
		logger.Info("Listing sessions")

	case "COMPONENT_LIST":
		logger.Info("Listing components for session:", cmd.Payload)

	case "COMPONENT_GET":
		logger.Info("Getting component:", cmd.Payload)
	case "COMPONENT_DESCRIBE":
		logger.Info("Describing component:", cmd.Payload)

	case "HEALTH_CHECK":
		handleHealthCheck(cmd.Payload, conn)

	default:
		logger.Warn("Unknown command:", cmd.Type)
	}
}
