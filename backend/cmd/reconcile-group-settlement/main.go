package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	expense "expense-tracker/backend/services/expense"
	expensestore "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func main() {
	groupID := flag.String("group-id", "", "group UUID to reconcile")
	flag.Parse()
	if _, err := uuid.Parse(*groupID); err != nil {
		fmt.Fprintln(os.Stderr, "group-id must be a valid UUID")
		os.Exit(2)
	}

	storage, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to initialize database connection")
		os.Exit(1)
	}
	defer storage.Close()
	if err := storage.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "unable to connect to database")
		os.Exit(1)
	}

	store := expensestore.NewStore(storage)
	var result expense.SettlementReconciliation
	err = store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if _, err := transactionStore.LockGroupCurrency(*groupID); err != nil {
			return err
		}
		var reconcileErr error
		result, reconcileErr = expense.ReconcileGroupSettlement(transactionStore, expense.NewController(), *groupID)
		return reconcileErr
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to reconcile group settlement")
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, "unable to encode reconciliation result")
		os.Exit(1)
	}
}
