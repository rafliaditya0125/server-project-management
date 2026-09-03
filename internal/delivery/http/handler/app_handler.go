package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/response"
)

type AppHandler struct {
	appUsecase     domain.AppUsecase
	serviceUsecase domain.ServiceUsecase
}

func NewAppHandler(appUsecase domain.AppUsecase, serviceUsecase domain.ServiceUsecase) *AppHandler {
	return &AppHandler{
		appUsecase:     appUsecase,
		serviceUsecase: serviceUsecase,
	}
}

func (h *AppHandler) ListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.appUsecase.List()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list applications", err.Error())
		return
	}
	response.Success(w, "Applications retrieved successfully", apps)
}

func (h *AppHandler) GetApp(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	app, err := h.appUsecase.GetByName(name)
	if err != nil {
		if err == domain.ErrAppNotFound {
			response.Error(w, http.StatusNotFound, "Application not found", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to get application", err.Error())
		return
	}
	response.Success(w, "Application retrieved successfully", app)
}

func (h *AppHandler) CreateApp(w http.ResponseWriter, r *http.Request) {
	var dto domain.CreateAppDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON payload", err.Error())
		return
	}

	app, err := h.appUsecase.Create(&dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Failed to create application", err.Error())
		return
	}

	response.Created(w, "Application created successfully", app)
}

func (h *AppHandler) DeleteApp(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var dto domain.DeleteAppDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Name = name

	if err := h.appUsecase.Delete(&dto); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete application", err.Error())
		return
	}

	response.Success(w, "Application deleted successfully", nil)
}

func (h *AppHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	lines := 100
	if l := r.URL.Query().Get("lines"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			lines = val
		}
	}

	logs, err := h.appUsecase.GetLogs(name, lines)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get application logs", err.Error())
		return
	}

	response.Success(w, "Logs retrieved successfully", map[string]string{"logs": logs})
}

type ManageRequest struct {
	Action domain.ServiceAction `json:"action"`
}

func (h *AppHandler) ManageService(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req ManageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	out, err := h.serviceUsecase.Manage(name, req.Action)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to manage service", err.Error())
		return
	}

	response.Success(w, "Service action executed successfully", map[string]string{"output": out})
}
