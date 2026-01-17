package route

import "expense-tracker/backend/types"

type baseRefreshStore struct {
	CreateRefreshTokenFn func(token types.RefreshToken) error
	GetRefreshTokenByIDFn func(id string) (*types.RefreshToken, error)
	RevokeRefreshTokenFn func(id string) error
}

func (m *baseRefreshStore) CreateRefreshToken(token types.RefreshToken) error {
	if m.CreateRefreshTokenFn != nil {
		return m.CreateRefreshTokenFn(token)
	}
	return nil
}

func (m *baseRefreshStore) GetRefreshTokenByID(id string) (*types.RefreshToken, error) {
	if m.GetRefreshTokenByIDFn != nil {
		return m.GetRefreshTokenByIDFn(id)
	}
	return nil, types.ErrInvalidToken
}

func (m *baseRefreshStore) RevokeRefreshToken(id string) error {
	if m.RevokeRefreshTokenFn != nil {
		return m.RevokeRefreshTokenFn(id)
	}
	return nil
}

func refreshStoreMock() *baseRefreshStore {
	return &baseRefreshStore{}
}
