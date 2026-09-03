package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
)

type JSONConfigRepository struct {
	mu       sync.RWMutex
	filePath string
}

func NewJSONConfigRepository(filePath string) *JSONConfigRepository {
	repo := &JSONConfigRepository{filePath: filePath}
	_ = repo.ensureFile()
	return repo
}

func (r *JSONConfigRepository) ensureFile() error {
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		return os.WriteFile(r.filePath, []byte("{}"), 0644)
	}
	return nil
}

func (r *JSONConfigRepository) Get() (*domain.SystemConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.ensureFile(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg domain.SystemConfig
	if len(data) == 0 {
		return &cfg, nil
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config json: %w", err)
	}

	return &cfg, nil
}

func (r *JSONConfigRepository) Save(cfg *domain.SystemConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureFile(); err != nil {
		return err
	}

	marshaled, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", r.filePath, os.Getpid())
	if err := os.WriteFile(tmpFile, marshaled, 0644); err != nil {
		return fmt.Errorf("failed to write tmp config file: %w", err)
	}

	if err := os.Rename(tmpFile, r.filePath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

func (r *JSONConfigRepository) GetValue(key string, defaultVal string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.ensureFile(); err != nil {
		return defaultVal, nil
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return defaultVal, nil
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return defaultVal, nil
	}

	if val, ok := rawMap[key]; ok && val != nil {
		return fmt.Sprintf("%v", val), nil
	}

	return defaultVal, nil
}

func (r *JSONConfigRepository) SaveValue(key string, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureFile(); err != nil {
		return err
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	rawMap := make(map[string]interface{})
	if len(data) > 0 {
		_ = json.Unmarshal(data, &rawMap)
	}

	rawMap[key] = value

	marshaled, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", r.filePath, os.Getpid())
	if err := os.WriteFile(tmpFile, marshaled, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, r.filePath); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}

	return nil
}
