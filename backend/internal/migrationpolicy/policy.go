package migrationpolicy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	SchemaVersion   = 1
	BaselineVersion = 35
	ManifestName    = "manifest.json"
)

var (
	migrationPattern = regexp.MustCompile(`^(\d{6})_(.+)\.(up|down)\.sql$`)
	commentPattern   = regexp.MustCompile(`(?s)/\*.*?\*/|--[^\n]*`)
	categoryPatterns = []struct {
		pattern    *regexp.Regexp
		categories []string
	}{
		{regexp.MustCompile(`(?i)\bDROP\s+`), []string{"drop"}},
		{regexp.MustCompile(`(?is)\bRENAME\b.*?\bTO\b`), []string{"rename"}},
		{regexp.MustCompile(`(?is)\bALTER\s+COLUMN\b.*?\bTYPE\b`), []string{"type_change"}},
		{regexp.MustCompile(`(?i)\b(ADD\s+CONSTRAINT|SET\s+NOT\s+NULL|VALIDATE\s+CONSTRAINT)\b`), []string{"constraint_tightening"}},
		{regexp.MustCompile(`(?i)\bCREATE\s+(UNIQUE\s+)?INDEX\b`), []string{"index"}},
		{regexp.MustCompile(`(?i)\bTRUNCATE(?:\s+TABLE)?\b`), []string{"data_rewrite"}},
		{regexp.MustCompile(`(?is)\b(UPDATE|DELETE\s+FROM|INSERT\s+INTO.*?SELECT)\b`), []string{"backfill", "data_rewrite"}},
	}
	allowedCategories = map[string]bool{
		"additive":              true,
		"backfill":              true,
		"constraint_tightening": true,
		"data_rewrite":          true,
		"rename":                true,
		"drop":                  true,
		"type_change":           true,
		"index":                 true,
	}
)

type Manifest struct {
	SchemaVersion   int              `json:"schemaVersion"`
	BaselineVersion int              `json:"baselineVersion"`
	Migrations      []MigrationEntry `json:"migrations"`
}

type MigrationEntry struct {
	Version    uint           `json:"version"`
	Name       string         `json:"name"`
	Categories []string       `json:"categories"`
	Deployment string         `json:"deployment"`
	Backfill   BackfillPolicy `json:"backfill"`
	Locking    LockingPolicy  `json:"locking"`
	Rollback   RollbackPolicy `json:"rollback"`
}

type BackfillPolicy struct {
	Mode      string `json:"mode"`
	Resumable *bool  `json:"resumable"`
	Notes     string `json:"notes"`
}

type LockingPolicy struct {
	Risk  string `json:"risk"`
	Notes string `json:"notes"`
}

type RollbackPolicy struct {
	Application  string `json:"application"`
	Schema       string `json:"schema"`
	DataLossRisk *bool  `json:"dataLossRisk"`
	Notes        string `json:"notes"`
}

type migrationFile struct {
	name string
	sql  string
}

func LoadDirectory(directory string) (Manifest, error) {
	manifest, err := loadManifest(filepath.Join(directory, ManifestName))
	if err != nil {
		return Manifest{}, err
	}
	files, err := loadMigrationFiles(directory)
	if err != nil {
		return Manifest{}, err
	}
	if _, ok := files[BaselineVersion]; !ok {
		return Manifest{}, fmt.Errorf("migration baseline %06d SQL files are missing", BaselineVersion)
	}
	if err := validateManifest(manifest, files); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (manifest Manifest) ValidatePending(current uint, dirty bool) error {
	if dirty {
		return fmt.Errorf("database migration state is dirty: version=%06d dirty=true", current)
	}
	var blocked []string
	for _, entry := range manifest.Migrations {
		if entry.Version > current && entry.Deployment != "online" {
			blocked = append(blocked, fmt.Sprintf("%06d_%s", entry.Version, entry.Name))
		}
	}
	if len(blocked) > 0 {
		return fmt.Errorf(
			"normal deployment rejects maintenance-required pending migrations: %s",
			strings.Join(blocked, ", "),
		)
	}
	return nil
}

func (manifest Manifest) HasMaintenanceRequired() bool {
	for _, entry := range manifest.Migrations {
		if entry.Deployment != "online" {
			return true
		}
	}
	return false
}

func loadManifest(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open migration manifest: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode migration manifest: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode migration manifest: multiple JSON values")
		}
		return fmt.Errorf("decode migration manifest trailing data: %w", err)
	}
	return nil
}

func loadMigrationFiles(directory string) (map[int]map[string]migrationFile, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read migration directory: %w", err)
	}
	files := make(map[int]map[string]migrationFile)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		matches := migrationPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename: %s", entry.Name())
		}
		var version int
		if _, err := fmt.Sscanf(matches[1], "%d", &version); err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", matches[1], err)
		}
		direction := matches[3]
		if files[version] == nil {
			files[version] = make(map[string]migrationFile)
		}
		if _, exists := files[version][direction]; exists {
			return nil, fmt.Errorf("migration version %06d has duplicate %s files", version, direction)
		}
		content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		files[version][direction] = migrationFile{name: matches[2], sql: string(content)}
	}
	for version, directions := range files {
		up, upOK := directions["up"]
		down, downOK := directions["down"]
		if !upOK || !downOK {
			return nil, fmt.Errorf("migration version %06d must have matching up and down SQL files", version)
		}
		if up.name != down.name {
			return nil, fmt.Errorf("migration version %06d has mismatched up and down names", version)
		}
	}
	return files, nil
}

func validateManifest(manifest Manifest, files map[int]map[string]migrationFile) error {
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("migration manifest schemaVersion must be %d", SchemaVersion)
	}
	if manifest.BaselineVersion != BaselineVersion {
		return fmt.Errorf("migration manifest baselineVersion must be %d", BaselineVersion)
	}
	if manifest.Migrations == nil {
		return errors.New("migration manifest migrations must be an array")
	}

	seen := make(map[int]bool)
	previous := BaselineVersion
	for index, entry := range manifest.Migrations {
		if err := validateEntry(entry, index); err != nil {
			return err
		}
		version := int(entry.Version)
		if seen[version] {
			return fmt.Errorf("migration version %06d is duplicated", version)
		}
		if version <= previous {
			return errors.New("migration manifest versions must be strictly increasing")
		}
		directions, ok := files[version]
		if !ok {
			return fmt.Errorf("manifest entry %06d_%s has no matching SQL files", version, entry.Name)
		}
		if directions["up"].name != entry.Name {
			return fmt.Errorf(
				"manifest name %q does not match migration file name %q for version %06d",
				entry.Name,
				directions["up"].name,
				version,
			)
		}
		if err := validateSQLConsistency(entry, directions["up"].sql); err != nil {
			return err
		}
		seen[version] = true
		previous = version
	}

	var missing []int
	for version := range files {
		if version > BaselineVersion && !seen[version] {
			missing = append(missing, version)
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		rendered := make([]string, len(missing))
		for index, version := range missing {
			rendered[index] = fmt.Sprintf("%06d", version)
		}
		return fmt.Errorf("post-baseline migrations are missing manifest entries: %s", strings.Join(rendered, ", "))
	}
	return nil
}

func validateEntry(entry MigrationEntry, index int) error {
	label := fmt.Sprintf("migrations[%d]", index)
	if entry.Version <= BaselineVersion {
		return fmt.Errorf("%s.version must be greater than %d", label, BaselineVersion)
	}
	if strings.TrimSpace(entry.Name) == "" {
		return fmt.Errorf("%s.name must be non-empty", label)
	}
	if len(entry.Categories) == 0 {
		return fmt.Errorf("%s.categories must be non-empty", label)
	}
	seenCategories := make(map[string]bool)
	for _, category := range entry.Categories {
		if !allowedCategories[category] {
			return fmt.Errorf("%s.categories contains unsupported value %q", label, category)
		}
		if seenCategories[category] {
			return fmt.Errorf("%s.categories contains duplicate value %q", label, category)
		}
		seenCategories[category] = true
	}
	if entry.Deployment != "online" && entry.Deployment != "maintenance_required" {
		return fmt.Errorf("%s.deployment must be online or maintenance_required", label)
	}
	if err := validateBackfill(entry, label, seenCategories); err != nil {
		return err
	}
	if !slices.Contains([]string{"low", "medium", "high"}, entry.Locking.Risk) {
		return fmt.Errorf("%s.locking.risk must be low, medium, or high", label)
	}
	if strings.TrimSpace(entry.Locking.Notes) == "" {
		return fmt.Errorf("%s.locking.notes must be non-empty", label)
	}
	if !slices.Contains([]string{"compatible", "follow_up_required"}, entry.Rollback.Application) {
		return fmt.Errorf("%s.rollback.application is invalid", label)
	}
	if !slices.Contains([]string{"not_required", "manual_only", "unsafe"}, entry.Rollback.Schema) {
		return fmt.Errorf("%s.rollback.schema is invalid", label)
	}
	if entry.Rollback.DataLossRisk == nil {
		return fmt.Errorf("%s.rollback.dataLossRisk must be a boolean", label)
	}
	if strings.TrimSpace(entry.Rollback.Notes) == "" {
		return fmt.Errorf("%s.rollback.notes must be non-empty", label)
	}
	return nil
}

func validateBackfill(entry MigrationEntry, label string, categories map[string]bool) error {
	if entry.Backfill.Mode != "none" && entry.Backfill.Mode != "bounded" {
		return fmt.Errorf("%s.backfill.mode must be none or bounded", label)
	}
	if entry.Backfill.Resumable == nil {
		return fmt.Errorf("%s.backfill.resumable must be a boolean", label)
	}
	if strings.TrimSpace(entry.Backfill.Notes) == "" {
		return fmt.Errorf("%s.backfill.notes must be non-empty", label)
	}
	if entry.Backfill.Mode == "bounded" && !*entry.Backfill.Resumable {
		return fmt.Errorf("%s bounded backfill must be resumable", label)
	}
	if entry.Backfill.Mode == "none" && *entry.Backfill.Resumable {
		return fmt.Errorf("%s backfill mode none cannot be resumable", label)
	}
	if (categories["backfill"] || categories["data_rewrite"]) && entry.Backfill.Mode != "bounded" {
		return fmt.Errorf("%s backfill or data_rewrite category requires bounded mode", label)
	}
	return nil
}

func validateSQLConsistency(entry MigrationEntry, sql string) error {
	normalized := commentPattern.ReplaceAllString(sql, " ")
	for _, requirement := range categoryPatterns {
		if requirement.pattern.MatchString(normalized) && !containsAny(entry.Categories, requirement.categories) {
			return fmt.Errorf(
				"migration %06d_%s SQL requires one of categories %s",
				entry.Version,
				entry.Name,
				strings.Join(requirement.categories, ", "),
			)
		}
	}
	return nil
}

func containsAny(values, candidates []string) bool {
	for _, candidate := range candidates {
		if slices.Contains(values, candidate) {
			return true
		}
	}
	return false
}
