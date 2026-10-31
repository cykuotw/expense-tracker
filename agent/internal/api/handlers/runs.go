package handlers

import (
	"net/http"
	"uuid"
)

func (handler *Handler) HandleListRuns(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.application.ListRuns(request.Context(), request.PathValue("conversation"))
	writeResult(writer, items, err)
}

func (handler *Handler) HandleReadRunOutput(writer http.ResponseWriter, request *http.Request) {
	after := request.URL.Query().Get("after")
	if after != "" {
		if _, err := uuid.Parse(after); err != nil {
			http.Error(writer, "invalid cursor", http.StatusBadRequest)
			return
		}
	}
	items, err := handler.application.ReadRunOutput(request.Context(), request.PathValue("conversation"), request.PathValue("run"), after)
	writeResult(writer, items, err)
}
