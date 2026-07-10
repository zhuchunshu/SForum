package webreleaseruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func writeJSONAtomic(target string, value any) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := target + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("write temporary JSON: %w", err)
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary JSON: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync temporary JSON: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary JSON: %w", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		return fmt.Errorf("publish JSON: %w", err)
	}
	directory, err := os.Open(filepath.Dir(target))
	if err != nil {
		return fmt.Errorf("open JSON directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync JSON directory: %w", err)
	}
	return nil
}
