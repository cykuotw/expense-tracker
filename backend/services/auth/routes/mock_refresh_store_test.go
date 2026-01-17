package route

import "expense-tracker/backend/types"

type mockRefreshStore struct{}

func (m *mockRefreshStore) CreateRefreshToken(token types.RefreshToken) error {
	return nil
}

func (m *mockRefreshStore) GetRefreshTokenByID(id string) (*types.RefreshToken, error) {
	return nil, types.ErrInvalidToken
}

func (m *mockRefreshStore) RevokeRefreshToken(id string) error {
	return nil
}
