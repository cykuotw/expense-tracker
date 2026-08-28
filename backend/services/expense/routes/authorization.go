package expense

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (h *Handler) validateGroupParticipants(groupID uuid.UUID, userIDs ...uuid.UUID) error {
	if h.groupStore == nil {
		return nil
	}

	for _, userID := range userIDs {
		member, err := h.groupStore.CheckGroupUserPairExist(groupID.String(), userID.String())
		if err != nil {
			return err
		}
		if !member {
			return types.ErrGroupParticipantNotAllowed
		}
	}

	return nil
}
