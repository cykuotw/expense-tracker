package errornotifier

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/aws/aws-lambda-go/events"
)

const (
	maxCompressedPayloadBytes   = 256 * 1024
	maxDecompressedPayloadBytes = 1024 * 1024
	maxJSONDepth                = 32
)

func decodeSubscription(event events.CloudwatchLogsEvent) (events.CloudwatchLogsData, error) {
	encoded := event.AWSLogs.Data
	if encoded == "" || len(encoded) > base64.StdEncoding.EncodedLen(maxCompressedPayloadBytes) {
		return events.CloudwatchLogsData{}, permanent("decode subscription payload")
	}

	compressed, err := io.ReadAll(io.LimitReader(
		base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(encoded)),
		maxCompressedPayloadBytes+1,
	))
	if err != nil || len(compressed) > maxCompressedPayloadBytes {
		return events.CloudwatchLogsData{}, permanent("decode subscription payload")
	}

	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return events.CloudwatchLogsData{}, permanent("decompress subscription payload")
	}
	decompressed, readErr := io.ReadAll(io.LimitReader(reader, maxDecompressedPayloadBytes+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || len(decompressed) > maxDecompressedPayloadBytes {
		return events.CloudwatchLogsData{}, permanent("decompress subscription payload")
	}
	if err := validateJSONDepth(decompressed); err != nil {
		return events.CloudwatchLogsData{}, permanent("validate subscription payload")
	}

	var data events.CloudwatchLogsData
	if err := json.Unmarshal(decompressed, &data); err != nil {
		return events.CloudwatchLogsData{}, permanent("parse subscription payload")
	}
	return data, nil
}

func validateJSONDepth(value []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(value))
	depth := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			continue
		}
		switch delimiter {
		case '{', '[':
			depth++
			if depth > maxJSONDepth {
				return permanent("validate JSON depth")
			}
		case '}', ']':
			depth--
		}
	}
}
