package handlers

import (
	"encoding/json"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/health"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Target  string `json:"target"`
	Message string `json:"message"`
}

func HandleHealthCheck(target string, conn net.Conn, sessionStore *core.SessionStore) {
	var response HealthResponse

	switch target {

	case "all":
		err := health.CheckAll()
		if err != nil {
			response = HealthResponse{
				Status:  "ERROR",
				Target:  "all",
				Message: err.Error(),
			}
			logger.Error("Health check failed:", err)
		}

		response = HealthResponse{
			Status:  "OK",
			Target:  "all",
			Message: "All subsystems healthy",
		}

	case "data":
		err := health.DataHealthCheck()
		if err != nil {
			response = HealthResponse{
				Status:  "ERROR",
				Target:  "data",
				Message: err.Error(),
			}
			logger.Error("Data health check failed:", err)
		}

		response = HealthResponse{
			Status:  "OK",
			Target:  "data",
			Message: "Data quality and availability healthy",
		}

	case "sessions":
		err := health.SessionsHealthCheck()
		if err != nil {
			response = HealthResponse{
				Status:  "ERROR",
				Target:  "sessions",
				Message: err.Error(),
			}
			logger.Error("Session manager health check failed:", err)
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
