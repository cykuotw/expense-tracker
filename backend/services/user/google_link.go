package user

import (
	"errors"
	"expense-tracker/backend/services/auth"
	googleAuth "expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func googleLinkBearerToken(header string) (string, error) {
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

func (h *HandlerProtected) handleGoogleLinkUpstreamVerified(c *gin.Context) {
	claims, err := googleAuth.VerifiedClaimsFromContext(c.Request.Context())
	if err != nil {
		writeGoogleLinkError(c, err)
		return
	}
	h.finishGoogleLink(c, claims)
}

func (h *HandlerProtected) handleGoogleLinkInProcess(c *gin.Context) {
	rawToken, err := googleLinkBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		writeGoogleLinkError(c, err)
		return
	}
	claims, err := h.googleVerifier.VerifyGoogleIDToken(c.Request.Context(), rawToken)
	if err != nil {
		writeGoogleLinkError(c, err)
		return
	}
	h.finishGoogleLink(c, claims)
}

func (h *HandlerProtected) finishGoogleLink(c *gin.Context, claims *types.VerifiedGoogleClaims) {
	if claims == nil || strings.TrimSpace(claims.Subject) == "" {
		writeGoogleLinkError(c, types.ErrMissingGoogleSubject)
		return
	}
	if auth.NormalizeEmail(claims.Email) == "" {
		writeGoogleLinkError(c, types.ErrMissingGoogleEmail)
		return
	}
	if claims.EmailVerified == nil || !*claims.EmailVerified {
		writeGoogleLinkError(c, types.ErrGoogleEmailNotVerified)
		return
	}

	var payload types.LinkGoogleAccountPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}

	userID, ok := accountUserID(c)
	if !ok {
		return
	}
	if err := h.store.LinkGoogleIdentity(
		c.Request.Context(),
		userID,
		payload.CurrentPassword,
		strings.TrimSpace(claims.Subject),
		auth.NormalizeEmail(claims.Email),
		currentRefreshID(c, userID),
	); err != nil {
		writeGoogleLinkError(c, err)
		return
	}
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		writeGoogleLinkError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, accountResponse(user))
}

func writeGoogleLinkError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, types.ErrMissingAuthorizationHeader),
		errors.Is(err, types.ErrInvalidAuthorizationHeader),
		errors.Is(err, types.ErrMissingBearerToken),
		errors.Is(err, types.ErrGoogleClaimsUnavailable),
		errors.Is(err, types.ErrInvalidGoogleIDToken),
		errors.Is(err, types.ErrInvalidGoogleIssuer):
		utils.WriteError(c, http.StatusUnauthorized, err)
	case errors.Is(err, types.ErrMissingGoogleSubject),
		errors.Is(err, types.ErrMissingGoogleEmail),
		errors.Is(err, types.ErrGoogleEmailNotVerified):
		utils.WriteError(c, http.StatusBadRequest, err)
	default:
		writeAccountError(c, err)
	}
}
