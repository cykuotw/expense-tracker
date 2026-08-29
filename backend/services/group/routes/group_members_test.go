package group

import (
	"encoding/json"
	"testing"

	"expense-tracker/backend/types"

	"github.com/stretchr/testify/assert"
)

func TestGroupMembersForUserSerializesAnEmptyStoreResultAsArray(t *testing.T) {
	members := groupMembersForUser(nil, "current-user")

	body, err := json.Marshal(members)
	assert.NoError(t, err)
	assert.JSONEq(t, "[]", string(body))
}

func TestGroupMembersForUserDoesNotAppendAnEmptyCurrentUser(t *testing.T) {
	users := []*types.User{{Username: "Other"}}

	members := groupMembersForUser(users, "current-user")

	assert.Len(t, members, 1)
	assert.Equal(t, "Other", members[0].Username)
}
