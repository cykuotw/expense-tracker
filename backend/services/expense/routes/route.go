package expense

import (
	"context"
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/services/middleware/validation"
	"expense-tracker/backend/types"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store      types.ExpenseStore
	userStore  types.UserStore
	groupStore types.GroupStore

	controller            types.ExpenseController
	receiptObjects        *receiptObjectStore
	receiptSecret         []byte
	receiptStorageEnabled bool
	receiptView           func(context.Context, string) (string, error)
	evidenceCleanup       func(context.Context, string) error
	evidenceReconcile     func(context.Context, string) error
}

func WithEvidenceCleanup(cleanup func(context.Context, string) error, reconcile ...func(context.Context, string) error) func(*Handler) {
	return func(h *Handler) {
		h.evidenceCleanup = cleanup
		if len(reconcile) > 0 {
			h.evidenceReconcile = reconcile[0]
		}
	}
}

func NewHandler(store types.ExpenseStore, userStore types.UserStore, groupStore types.GroupStore, controller types.ExpenseController, options ...func(*Handler)) *Handler {
	h := &Handler{
		store:      store,
		userStore:  userStore,
		groupStore: groupStore,

		controller: controller,
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func WithReceiptStorage(client *s3.Client, bucket string, secret []byte, enabled bool) func(*Handler) {
	return func(h *Handler) {
		if client != nil && bucket != "" && len(secret) >= 32 {
			h.receiptObjects = &receiptObjectStore{client: client, bucket: bucket}
			h.receiptSecret = append([]byte(nil), secret...)
			h.receiptStorageEnabled = enabled
			presigner := s3.NewPresignClient(client)
			h.receiptView = func(ctx context.Context, key string) (string, error) {
				cache := "private, no-store"
				contentType := "image/jpeg"
				result, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key, ResponseCacheControl: &cache, ResponseContentType: &contentType}, func(o *s3.PresignOptions) { o.Expires = 45 * time.Second })
				if err != nil {
					return "", err
				}
				return result.URL, nil
			}
		}
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.Use(extractors.ExtractUserIdFromJWT())

	router.POST("/create_expense",
		extractors.ExtractExpensePayload(),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleCreateExpense)
	router.GET("/expense_list/:groupId",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetExpenseList)
	router.GET("/expense_list/:groupId/:page",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetExpenseList)
	router.GET("/group_overview/:groupId/:page",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetGroupOverview)
	router.GET("/expense_types", h.handleGetExpenseType)
	router.GET("/expense_create_options", h.handleGetCreateExpenseOptions)
	router.GET("/expense/:expenseId/edit-options",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetEditExpenseOptions)
	router.GET("/expense/:expenseId",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetExpenseDetail)
	router.POST("/expense/:expenseId/receipt/view",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore), h.handleReceiptView)
	router.PUT("/expense/:expenseId/receipt",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleReceiptMutation)
	router.POST("/expense/:expenseId/receipt/reconcile",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleReceiptReconcile)
	router.POST("/expense/:expenseId/receipt/retry",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleReceiptRetry)
	router.PUT("/expense/:expenseId",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		extractors.ExtractExpenseUpdatePayload(),
		h.handleUpdateExpense)
	router.PUT("/delete_expense/:expenseId",
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleDeleteExpense)
	router.PUT("/settle_expense/:groupId",
		h.observeSettlementGuard("group"),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleSettleExpense)
	router.GET("/balance/:groupId",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetUnsettledBalance)
	router.POST("/settle_balance/:groupId/:balanceId",
		h.observeSettlementGuard("balance"),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleSettleBalance)
}
