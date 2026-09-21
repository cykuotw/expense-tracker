package frontendreport

import (
	"encoding/json"
	"errors"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/utils"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxReportBodyBytes = 128
	reportCooldown     = 10 * time.Minute
	replayTTL          = 24 * time.Hour
	maxTrackedReports  = 2048
)

var errInvalidReport = errors.New("invalid frontend error report")

type reportPayload struct {
	OccurrenceID string `json:"occurrenceId"`
}

type limiter struct {
	mu          sync.Mutex
	lastByUser  map[string]time.Time
	seenReports map[string]time.Time
	now         func() time.Time
}

type Handler struct {
	limiter *limiter
}

func NewHandler() *Handler {
	return &Handler{limiter: &limiter{
		lastByUser:  make(map[string]time.Time),
		seenReports: make(map[string]time.Time),
		now:         time.Now,
	}}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/observability/frontend-render-error", h.handleRenderError)
}

func (h *Handler) handleRenderError(c *gin.Context) {
	contentType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || contentType != "application/json" {
		utils.WriteError(c, http.StatusUnsupportedMediaType, errInvalidReport)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxReportBodyBytes)
	var payload reportPayload
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, errInvalidReport)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		utils.WriteError(c, http.StatusBadRequest, errInvalidReport)
		return
	}

	occurrenceID, err := uuid.Parse(payload.OccurrenceID)
	if err != nil || occurrenceID.Version() != 4 || occurrenceID.Variant() != uuid.RFC4122 {
		utils.WriteError(c, http.StatusBadRequest, errInvalidReport)
		return
	}
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, errInvalidReport)
		return
	}

	if !h.limiter.allow(userID, occurrenceID.String()) {
		c.Status(http.StatusNoContent)
		return
	}

	observability.Logger(c).Error(observability.EventFrontendRenderError,
		slog.Bool("alertable", true),
		slog.String("method", http.MethodPost),
		slog.String("route", observability.MatchedRoute(c)),
		slog.Int("status", http.StatusInternalServerError),
		slog.String("error_code", "frontend_render_error"),
		slog.String("error_category", "frontend_render"),
		slog.String("occurrence_id", occurrenceID.String()),
	)
	c.Status(http.StatusAccepted)
}

func (l *limiter) allow(userID, occurrenceID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for key, last := range l.lastByUser {
		if now.Sub(last) >= reportCooldown {
			delete(l.lastByUser, key)
		}
	}
	for key, expiresAt := range l.seenReports {
		if !expiresAt.After(now) {
			delete(l.seenReports, key)
		}
	}
	if expiresAt, seen := l.seenReports[occurrenceID]; seen && expiresAt.After(now) {
		return false
	}
	if last, exists := l.lastByUser[userID]; exists && now.Sub(last) < reportCooldown {
		return false
	}
	if len(l.seenReports) >= maxTrackedReports {
		return false
	}

	l.lastByUser[userID] = now
	l.seenReports[occurrenceID] = now.Add(replayTTL)
	return true
}
