package invitation_test

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/invitation"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationStorePersistsOnlyTokenHashAndExchangesItForSession(t *testing.T) {
	dbConn := openTestDB(t)
	store := invitation.NewStore(dbConn)

	inviterID := uuid.New()
	setupUser(t, dbConn, inviterID)
	t.Cleanup(func() { cleanUser(t, dbConn, inviterID) })

	rawToken := "test-invitation-" + uuid.NewString()
	invitationID := uuid.New()
	require.NoError(t, store.CreateInvitation(types.Invitation{
		ID:        invitationID,
		TokenHash: auth.HashToken(rawToken),
		Email:     "invitee@test.com",
		InviterID: inviterID,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}))
	t.Cleanup(func() { cleanInvitation(t, dbConn, invitationID) })

	var storedHash string
	require.NoError(t, dbConn.QueryRow("SELECT token_hash FROM invitations WHERE id = $1", invitationID).Scan(&storedHash))
	assert.Equal(t, auth.HashToken(rawToken), storedHash)
	assert.NotEqual(t, rawToken, storedHash)
	var active bool
	require.NoError(t, dbConn.QueryRow("SELECT expires_at > NOW() FROM invitations WHERE id = $1", invitationID).Scan(&active))
	require.True(t, active)

	registrationSession := "registration-session-" + uuid.NewString()
	exchanged, err := store.ExchangeInvitation(rawToken, registrationSession)
	require.NoError(t, err)
	assert.Equal(t, invitationID, exchanged.ID)
	assert.Equal(t, "invitee@test.com", exchanged.Email)

	var sessionHash string
	require.NoError(t, dbConn.QueryRow("SELECT registration_session_hash FROM invitations WHERE id = $1", invitationID).Scan(&sessionHash))
	assert.Equal(t, auth.HashToken(registrationSession), sessionHash)
	assert.NotEqual(t, registrationSession, sessionHash)
}

func TestRotateInvitationInvalidatesEarlierSecretAndRegistrationSession(t *testing.T) {
	dbConn := openTestDB(t)
	store := invitation.NewStore(dbConn)

	inviterID := uuid.New()
	setupUser(t, dbConn, inviterID)
	t.Cleanup(func() { cleanUser(t, dbConn, inviterID) })

	firstToken := "first-invitation-" + uuid.NewString()
	invitationID := uuid.New()
	require.NoError(t, store.CreateInvitation(types.Invitation{
		ID:        invitationID,
		TokenHash: auth.HashToken(firstToken),
		Email:     "invitee@test.com",
		InviterID: inviterID,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}))
	t.Cleanup(func() { cleanInvitation(t, dbConn, invitationID) })

	_, err := store.ExchangeInvitation(firstToken, "first-session-"+uuid.NewString())
	require.NoError(t, err)
	rotatedToken, err := store.RotateInvitationTokenByID(invitationID.String())
	require.NoError(t, err)
	assert.NotEqual(t, firstToken, rotatedToken)

	_, err = store.ExchangeInvitation(firstToken, "old-session-"+uuid.NewString())
	assert.ErrorIs(t, err, types.ErrInvitationInvalid)
	_, err = store.ExchangeInvitation(rotatedToken, "new-session-"+uuid.NewString())
	require.NoError(t, err)
}
