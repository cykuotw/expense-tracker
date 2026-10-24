// ocr-evidence replays saved Gemini responses offline. It has no database access.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"expense-tracker/backend/services/ocr"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "offline OCR replay failed; verify the action and input file")
		os.Exit(1)
	}
}

func run() error {
	action := flag.String("action", "replay", "replay (offline only)")
	file := flag.String("file", "", "local saved provider response or historical bundle")
	flag.Parse()
	if *action != "replay" {
		return errors.New("only offline replay is supported; database evidence cleanup is retired")
	}
	return replay(*file)
}

func replay(file string) error {
	input, err := os.Open(file)
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
