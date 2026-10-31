package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

const balanceRequestBody = `{"conversationId":"one","messageId":"one","text":"balance demo"}`

func localRequest(body string) *http.Request {
	request := httptest.NewRequest(
		"POST", "http://127.0.0.1:8090/chat", strings.NewReader(body),
	)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Content-Type", "application/json")
	return request
}
