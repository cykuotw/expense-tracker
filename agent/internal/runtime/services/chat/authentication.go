package chat

import (
	"context"
	"time"

	"expense-tracker/agent/internal/models"
)

func (service *Service) resolveOwner(ctx context.Context) (string, error) {
	if service.authenticate == nil {
		return "local-fixture", nil
	}
	actor, err := service.authenticate(ctx)
	if err != nil {
		return "", &models.Error{Code: models.PublicCode(err)}
	}
	if actor.ID == "" || !actor.ExpiresAt.After(time.Now()) {
		return "", &models.Error{Code: models.Unauthenticated}
	}
	return actor.ID, nil
}

// AuthenticatedOwner resolves a trusted owner for recorded history access.
func (service *Service) AuthenticatedOwner(ctx context.Context) (string, error) {
	if service.records == nil || service.authenticate == nil {
		return "", &models.Error{Code: models.Unavailable}
	}
	return service.resolveOwner(ctx)
}
