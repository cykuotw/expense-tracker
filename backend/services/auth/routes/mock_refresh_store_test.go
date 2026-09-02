package route

import "expense-tracker/backend/types"

type baseRefreshStore struct {
	CreateRefreshTokenFn       func(token types.RefreshToken) error
	GetRefreshTokenByIDFn      func(id string) (*types.RefreshToken, error)
	RotateRefreshTokenFn       func(id string, tokenHash string, successor types.RefreshToken) error
	RevokeRefreshTokenFamilyFn func(id string) error
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

func (m *baseRefreshStore) RotateRefreshToken(id string, tokenHash string, successor types.RefreshToken) error {
	if m.RotateRefreshTokenFn != nil {
		return m.RotateRefreshTokenFn(id, tokenHash, successor)
	}
	return nil
}

func (m *baseRefreshStore) RevokeRefreshTokenFamily(id string) error {
	if m.RevokeRefreshTokenFamilyFn != nil {
		return m.RevokeRefreshTokenFamilyFn(id)
	}
	return nil
}

func refreshStoreMock() *baseRefreshStore {
	return &baseRefreshStore{}
}
