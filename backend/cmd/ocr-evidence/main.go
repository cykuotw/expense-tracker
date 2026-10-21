// ocr-evidence is an operator-only bounded purge and offline adapter replay tool.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"expense-tracker/backend/services/ocr"
	"expense-tracker/backend/services/ocrimprovement"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "evidence operation incomplete; verify configuration and retry")
		os.Exit(1)
	}
}
func run() error {
	action := flag.String("action", "replay", "replay, request-account-purge, purge-account, purge-expense, or cleanup")
	file := flag.String("file", "", "local saved provider response or evidence bundle for offline replay")
	id := flag.String("id", "", "account or expense UUID for explicit purge")
	flag.Parse()
	if *action == "replay" {
		input, err := os.Open(*file)
		if err != nil {
			return err
		}
		defer input.Close()
		raw, err := io.ReadAll(io.LimitReader(input, 8*1024*1024+1))
		if err != nil || len(raw) > 8*1024*1024 {
			return errors.New("oversized replay file")
		}
		if len(raw) > 2 && raw[0] == 0x1f && raw[1] == 0x8b {
			reader, err := gzip.NewReader(bytes.NewReader(raw))
			if err != nil {
				return err
			}
			defer reader.Close()
			raw, err = io.ReadAll(io.LimitReader(reader, 8*1024*1024+1))
			if err != nil || len(raw) > 8*1024*1024 {
				return errors.New("oversized replay bundle")
			}
		}
		var bundle struct {
			Extraction *ocr.ExtractionEvidence `json:"extraction"`
		}
		_ = json.Unmarshal(raw, &bundle)
		if bundle.Extraction != nil {
			result, err := ocr.ReplayEvidence(*bundle.Extraction)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(ocr.PrepareDraft(result))
		}
		result, _, err := ocr.ReplayGeminiResponse(raw)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(ocr.PrepareDraft(result))
	}
	if *action != "cleanup" {
		if _, err := uuid.Parse(*id); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	database, err := db.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		return err
	}
	defer database.Close()
	s := &ocrimprovement.Service{DB: database, Bucket: os.Getenv("OCR_EVIDENCE_BUCKET")}
	if *action == "request-account-purge" {
		if err := s.RequestAccountPurge(ctx, *id); err != nil {
			return err
		}
		fmt.Println("purge_state=pending; rerun purge-account after two minutes")
		return nil
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	s.Objects = s3.NewFromConfig(awsCfg)
	switch *action {
	case "purge-account":
		err = s.PurgeAccount(ctx, *id)
	case "purge-expense":
		err = s.PurgeExpense(ctx, *id)
	case "cleanup":
		for _, query := range []struct {
			sql   string
			apply func(context.Context, string) error
		}{
			{`SELECT account_id::text FROM ocr_improvement_consent WHERE purge_pending ORDER BY account_id LIMIT 10`, s.PurgeAccount},
			{`SELECT DISTINCT expense_id::text FROM ocr_improvement_evidence WHERE state='deleting' AND expense_id IS NOT NULL ORDER BY expense_id::text LIMIT 10`, s.CleanupExpense},
		} {
			rows, queryErr := database.QueryContext(ctx, query.sql)
			if queryErr != nil {
				return queryErr
			}
			ids := []string{}
			for rows.Next() {
				var value string
				if scanErr := rows.Scan(&value); scanErr != nil {
					rows.Close()
					return scanErr
				}
				ids = append(ids, value)
			}
			queryErr = rows.Err()
			rows.Close()
			if queryErr != nil {
				return queryErr
			}
			for _, value := range ids {
				if applyErr := query.apply(ctx, value); applyErr != nil {
					err = applyErr
				}
			}
		}
		var pending bool
		if queryErr := database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ocr_improvement_consent WHERE purge_pending) OR EXISTS(SELECT 1 FROM ocr_improvement_evidence WHERE state='deleting')`).Scan(&pending); queryErr != nil {
			return queryErr
		}
		if pending {
			return errors.New("cleanup has remaining pending work")
		}
	default:
		return errors.New("unknown action")
	}
	if err == nil {
		fmt.Println("evidence_operation=complete")
	}
	return err
}
