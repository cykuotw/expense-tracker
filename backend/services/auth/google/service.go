package google

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServiceContract interface {
	ResolveUserFromClaims(claims *types.VerifiedGoogleClaims) (*types.User, error)
}

type Service struct {
	store      types.UserStore
	now        func() time.Time
	newUUID    func() uuid.UUID
	hashSecret func(string) (string, error)
}

func NewService(store types.UserStore) *Service {
	return &Service{
		store:      store,
		now:        time.Now,
		newUUID:    uuid.New,
		hashSecret: auth.HashPassword,
	}
}

func (s *Service) ResolveUserFromClaims(claims *types.VerifiedGoogleClaims) (*types.User, error) {
	if claims == nil || claims.Subject == "" {
		return nil, types.ErrMissingGoogleSubject
	}

	user, err := s.store.GetUserByExternalIdentity("google", claims.Subject)
	if err == nil {
		return user, nil
	}
	if err != types.ErrUserNotExist {
		return nil, err
	}

	if strings.TrimSpace(claims.Email) == "" {
		return nil, types.ErrMissingGoogleEmail
	}

	user, err = s.store.GetUserByEmail(claims.Email)
	if err == nil {
		return nil, types.ErrGoogleAccountConflict
	}
	if err != types.ErrUserNotExist {
		return nil, err
	}

	if claims.EmailVerified == nil || !*claims.EmailVerified {
		return nil, types.ErrGoogleEmailNotVerified
	}

	hashedPassword, err := s.hashSecret("external-google-login-disabled:" + s.newUUID().String())
	if err != nil {
		return nil, err
	}

	user = &types.User{
		ID:             s.newUUID(),
		Username:       nicknameFromClaims(claims),
		Nickname:       nicknameFromClaims(claims),
		Firstname:      claims.GivenName,
		Lastname:       claims.FamilyName,
		Email:          claims.Email,
		PasswordHashed: hashedPassword,
		ExternalType:   "google",
		ExternalID:     claims.Subject,
		CreateTime:     s.now(),
		IsActive:       true,
		Role:           "user",
	}

	if err := s.store.CreateUser(*user); err != nil {
		return nil, err
	}

	return user, nil
}

func nicknameFromClaims(claims *types.VerifiedGoogleClaims) string {
	if claims == nil {
		return "google-user"
	}
	if claims.GivenName != "" {
		return claims.GivenName
	}
	if claims.Name != "" {
		return claims.Name
	}
	if localPart, _, found := strings.Cut(claims.Email, "@"); found && localPart != "" {
		return sanitizeUsername(localPart)
	}
	return "google-user"
}

func sanitizeUsername(value string) string {
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, value)
	sanitized = strings.Trim(sanitized, "-_")
	if sanitized == "" {
		return "google-user"
	}
	if len(sanitized) > 32 {
		return sanitized[:32]
	}
	return sanitized
}
