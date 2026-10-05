package types

// GroupLifecycle separates restoration authority from ordinary editing rights.
type GroupLifecycle struct {
	IsActive             bool   `json:"isActive"`
	CanManageLifecycle   bool   `json:"canManageLifecycle"`
	CanArchive           bool   `json:"canArchive"`
	CanRestore           bool   `json:"canRestore"`
	ArchiveBlockedReason string `json:"archiveBlockedReason,omitempty"`
}

type GroupLifecycleStore interface {
	GetGroupLifecycle(groupID, userID string) (GroupLifecycle, error)
	GetArchivedGroups(userID, cursor string, limit int) (ArchivedGroupPage, error)
}

type ArchivedGroupPage struct {
	Groups     []GetGroupListResponse `json:"groups"`
	NextCursor string                 `json:"nextCursor"`
}

// ReadGroupLifecycle uses persisted accounting state in production. Legacy store
// implementations can expose lifecycle state without granting archive eligibility.
func ReadGroupLifecycle(store GroupStore, group *Group, userID string) (GroupLifecycle, error) {
	if lifecycleStore, ok := store.(GroupLifecycleStore); ok {
		return lifecycleStore.GetGroupLifecycle(group.ID.String(), userID)
	}
	creator := group.CreateByUser.String() == userID
	return GroupLifecycle{IsActive: group.IsActive, CanManageLifecycle: creator, CanRestore: creator && !group.IsActive}, nil
}
