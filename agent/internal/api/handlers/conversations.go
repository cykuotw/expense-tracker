package handlers

import "net/http"

func (handler *Handler) HandleListConversations(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.application.ListConversations(request.Context())
	writeResult(writer, items, err)
}

func (handler *Handler) HandleDeleteConversation(writer http.ResponseWriter, request *http.Request) {
	err := handler.application.DeleteConversation(request.Context(), request.PathValue("conversation"))
	writeResult(writer, struct {
		Deleted bool `json:"deleted"`
	}{Deleted: err == nil}, err)
}
