package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
)

type JSONAppRepository struct {
	mu       sync.RWMutex
	filePath string
}

func NewJSONAppRepository(filePath string) *JSONAppRepository {
	repo := &JSONAppRepository{filePath: filePath}
	_ = repo.ensureFile()
	return repo
}

func (r *JSONAppRepository) ensureFile() error {
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		return os.WriteFile(r.filePath, []byte("[]"), 0644)
	}
	return nil
}

func (r *JSONAppRepository) FindAll() ([]domain.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.ensureFile(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	var apps []domain.App
	if len(data) == 0 {
		return []domain.App{}, nil
	}

	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse registry json: %w", err)
	}

	return apps, nil
}

func (r *JSONAppRepository) FindByName(name string) (*domain.App, error) {
	apps, err := r.FindAll()
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.Name == name {
			return &app, nil
		}
	}

	return nil, domain.ErrAppNotFound
}

func (r *JSONAppRepository) Exists(name string) (bool, error) {
	_, err := r.FindByName(name)
	if err == nil {
		return true, nil
	}
	if err == domain.ErrAppNotFound {
		return false, nil
	}
	return false, err
}

func (r *JSONAppRepository) Save(newApp *domain.App) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureFile(); err != nil {
		return err
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var apps []domain.App
	if len(data) > 0 {
		_ = json.Unmarshal(data, &apps)
	}

	// Filter existing by name and append new
	var updated []domain.App
	for _, a := range apps {
		if a.Name != newApp.Name {
			updated = append(updated, a)
		}
	}
	updated = append(updated, *newApp)

	return r.atomicWrite(updated)
}

func (r *JSONAppRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureFile(); err != nil {
		return err
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var apps []domain.App
	if len(data) > 0 {
		_ = json.Unmarshal(data, &apps)
	}

	var updated []domain.App
	for _, a := range apps {
		if a.Name != name {
			updated = append(updated, a)
		}
	}

	return r.atomicWrite(updated)
}

func (r *JSONAppRepository) atomicWrite(apps []domain.App) error {
	marshaled, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode apps: %w", err)
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", r.filePath, os.Getpid())
	if err := os.WriteFile(tmpFile, marshaled, 0644); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}

	if err := os.Rename(tmpFile, r.filePath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to replace registry file: %w", err)
	}

	return nil
}
