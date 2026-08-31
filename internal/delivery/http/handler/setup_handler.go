package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/response"
)

type SetupHandler struct {
	setupUsecase domain.SetupUsecase
}

func NewSetupHandler(setupUsecase domain.SetupUsecase) *SetupHandler {
	return &SetupHandler{setupUsecase: setupUsecase}
}

func (h *SetupHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	cfg, osDistro, err := h.setupUsecase.GetStatus()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get setup status", err.Error())
		return
	}

	response.Success(w, "Setup status retrieved successfully", map[string]any{
		"os":     osDistro,
		"config": cfg,
	})
}

func (h *SetupHandler) ExecuteSetup(w http.ResponseWriter, r *http.Request) {
	var opts domain.SetupOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		opts = domain.SetupOptions{All: true}
	}

	if err := h.setupUsecase.Execute(&opts); err != nil {
		response.Error(w, http.StatusInternalServerError, "Setup execution failed", err.Error())
		return
	}

	response.Success(w, "Setup completed successfully", nil)
}

type ConfigureFastCGIRequest struct {
	OS       string         `json:"os"`
	Mode     domain.PhpMode `json:"mode"`
	SockPath string         `json:"sock_path"`
	Port     string         `json:"port"`
}

func (h *SetupHandler) ConfigureFastCGI(w http.ResponseWriter, r *http.Request) {
	var req ConfigureFastCGIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	cfg, err := h.setupUsecase.ConfigureFastCGI(req.OS, req.Mode, req.SockPath, req.Port)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to configure FastCGI", err.Error())
		return
	}

	response.Success(w, "FastCGI configured successfully", cfg)
}
