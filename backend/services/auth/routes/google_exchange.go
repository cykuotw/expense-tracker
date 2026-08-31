package route

import (
	"errors"
	googleAuth "expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func extractBearerToken(header string) (string, error) {
	if strings.TrimSpace(header) == "" {
		return "", types.ErrMissingAuthorizationHeader
	}

	if !strings.HasPrefix(header, "Bearer ") {
		return "", types.ErrInvalidAuthorizationHeader
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return "", types.ErrMissingBearerToken
	}

	return token, nil
}

func googleExchangeStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch {
	case errors.Is(err, types.ErrMissingAuthorizationHeader),
		errors.Is(err, types.ErrInvalidAuthorizationHeader),
		errors.Is(err, types.ErrMissingBearerToken),
		errors.Is(err, types.ErrGoogleClaimsUnavailable),
		errors.Is(err, types.ErrInvalidGoogleIDToken),
		errors.Is(err, types.ErrInvalidGoogleIssuer):
		return http.StatusUnauthorized
	case errors.Is(err, types.ErrMissingGoogleSubject),
		errors.Is(err, types.ErrMissingGoogleEmail),
		errors.Is(err, types.ErrGoogleEmailNotVerified):
		return http.StatusBadRequest
	case errors.Is(err, types.ErrGoogleAccountConflict):
		return http.StatusConflict
	case errors.Is(err, types.ErrAccountConflict):
		return http.StatusConflict
	case errors.Is(err, types.ErrInvitationRequired):
		return http.StatusForbidden
	case errors.Is(err, types.ErrAccountInactive):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (h *Handler) handleGoogleExchangeUpstreamVerified(c *gin.Context) error {
	claims, err := googleAuth.VerifiedClaimsFromContext(c.Request.Context())
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}

	return h.finishGoogleExchange(c, claims)
}

func (h *Handler) handleGoogleExchangeInProcess(c *gin.Context) error {
	rawToken, err := extractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}

	claims, err := h.googleVerifier.VerifyGoogleIDToken(c.Request.Context(), rawToken)
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}

	return h.finishGoogleExchange(c, claims)
}

func (h *Handler) finishGoogleExchange(c *gin.Context, claims *types.VerifiedGoogleClaims) error {
	user, err := h.googleService.ResolveUserFromClaims(claims)
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}

	if err := h.issueAuthSession(c, user); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	utils.WriteJSON(c, http.StatusOK, nil)
	return nil
}

func (h *Handler) handleGoogleRegisterUpstreamVerified(c *gin.Context) error {
	claims, err := googleAuth.VerifiedClaimsFromContext(c.Request.Context())
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}
	return h.finishGoogleRegister(c, claims)
}

func (h *Handler) handleGoogleRegisterInProcess(c *gin.Context) error {
	rawToken, err := extractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}
	claims, err := h.googleVerifier.VerifyGoogleIDToken(c.Request.Context(), rawToken)
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}
	return h.finishGoogleRegister(c, claims)
}

func (h *Handler) finishGoogleRegister(c *gin.Context, claims *types.VerifiedGoogleClaims) error {
	var payload types.RegisterGooglePayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		if errors.Is(err, types.ErrEmptyRequestBody) {
			return writeRegistrationError(c, types.ErrInvitationRequired)
		}
		utils.WriteError(c, http.StatusBadRequest, err)
		return err
	}
	user, err := h.googleService.PrepareUserFromClaims(claims)
	if err != nil {
		return writeGoogleExchangeError(c, err)
	}
	if h.registrationStore == nil {
		err := errors.New("registration store is unavailable")
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}
	registrationSession, err := c.Cookie(registrationSessionCookieName)
	if err != nil {
		return writeRegistrationError(c, types.ErrInvitationRequired)
	}
	if err := h.registrationStore.CreateInvitedUser(c.Request.Context(), registrationSession, *user); err != nil {
		return writeRegistrationError(c, err)
	}

	clearRegistrationSessionCookie(c)
	utils.WriteJSON(c, http.StatusCreated, nil)
	return nil
}

func writeGoogleExchangeError(c *gin.Context, err error) error {
	status := googleExchangeStatus(err)
	utils.WriteError(c, status, err)
	if status >= http.StatusInternalServerError {
		return errors.New("google authentication failed")
	}
	return err
}
