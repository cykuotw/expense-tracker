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

func writeGoogleExchangeError(c *gin.Context, err error) error {
	utils.WriteError(c, googleExchangeStatus(err), err)
	return err
}
