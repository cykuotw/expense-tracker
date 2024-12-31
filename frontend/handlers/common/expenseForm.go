package common

type ExpenseForm struct {
	GroupId       string    `form:"groupId" binding:"required"`
	Description   string    `form:"description" binding:"required"`
	Payer         string    `form:"payer" binding:"required"`
	ExpenseTypeID string    `form:"expenseType" binding:"required"`
	Total         float32   `form:"total" binding:"required"`
	Currency      string    `form:"currency" binding:"required"`
	SpliteRule    string    `form:"splitRule" binding:"required"`
	Ids           []string  `form:"ledger.id[]" binding:"required"`
	Borrowers     []string  `form:"ledger.borrower[]" binding:"required"`
	Shares        []float32 `form:"ledger.share[]" binding:"required"`
}
