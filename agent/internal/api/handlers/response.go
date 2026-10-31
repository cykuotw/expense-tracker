package handlers

import (
	"encoding/json/v2"
	"net/http"

	"expense-tracker/agent/internal/models"
)

func writeResult(writer http.ResponseWriter, value any, err error) {
	if err != nil {
		status := 503
		switch models.PublicCode(err) {
		case models.Unauthenticated:
			status = 401
		case models.Forbidden:
			status = 403
		case models.NotFound:
			status = 404
		case models.Validation:
			status = 400
		case models.Limit:
			status = 409
		}
		http.Error(writer, "request rejected", status)
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(writer, "response unavailable", 503)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(data)
}
