package chat

import (
	"uuid"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime/engine"
)

const (
	maxConversations        = 64
	maxConversationMessages = 64
	maxActiveRuns           = 4
)

type conversationKey struct {
	owner        string
	conversation string
}

type conversationTracker struct {
	activeRunID string
	messageIDs  map[string]bool
	runIDs      map[string]bool
	sessionID   string
}

// reserveRun keeps admission checks and state changes under the same lock.
func (service *Service) reserveRun(owner string, request models.ChatRequest) (models.Key, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	key := conversationKey{owner: owner, conversation: request.ConversationID}
	current := service.conversations[key]
	if current == nil {
		if len(service.conversations) >= maxConversations {
			return models.Key{}, &models.Error{Code: models.Limit}
		}
		current = &conversationTracker{messageIDs: map[string]bool{}, runIDs: map[string]bool{}, sessionID: uuid.New().String()}
		service.conversations[key] = current
	}
	atLimit := service.activeRunCount >= maxActiveRuns || len(current.messageIDs) >= maxConversationMessages
	if current.activeRunID != "" || atLimit || current.messageIDs[request.MessageID] || current.runIDs[request.RunID] {
		return models.Key{}, &models.Error{Code: models.Limit}
	}
	current.activeRunID = request.RunID
	current.messageIDs[request.MessageID] = true
	current.runIDs[request.RunID] = true
	service.activeRunCount++
	return models.Key{App: engine.AppName, Owner: owner, Session: current.sessionID}, nil
}

func (service *Service) releaseRun(owner string, request models.ChatRequest, allowRetry bool) {
	service.mu.Lock()
	defer service.mu.Unlock()

	current := service.conversations[conversationKey{owner: owner, conversation: request.ConversationID}]
	current.activeRunID = ""
	if allowRetry {
		delete(current.messageIDs, request.MessageID)
		delete(current.runIDs, request.RunID)
	}
	service.activeRunCount--
}

// IsRunning reports whether this process currently owns the given turn.
func (service *Service) IsRunning(owner, conversation, run string) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	current := service.conversations[conversationKey{owner: owner, conversation: conversation}]
	return current != nil && current.activeRunID != "" && current.activeRunID == run
}
