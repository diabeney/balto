package services

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/diabeney/balto/internal/api/core"
	"github.com/diabeney/balto/pkg/utils"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sm := core.GetServiceManager()
	if sm == nil {
		utils.WriteError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	services := sm.ListServices()
	utils.WriteJSON(w, http.StatusOK, services)
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sm := core.GetServiceManager()
	if sm == nil {
		utils.WriteError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	id := extractServiceID(r.URL.Path)
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "service ID required")
		return
	}

	svc, exists := sm.GetService(id)
	if !exists {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, svc)
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sm := core.GetServiceManager()
	if sm == nil {
		utils.WriteError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	var req core.ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Domain == "" || req.PathPrefix == "" || len(req.Ports) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "domain, path_prefix, and ports are required")
		return
	}

	svc, err := sm.AddService(req.Domain, req.PathPrefix, req.Ports)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, svc)
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sm := core.GetServiceManager()
	if sm == nil {
		utils.WriteError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	id := extractServiceID(r.URL.Path)
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "service ID required")
		return
	}

	var req core.ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Domain == "" || req.PathPrefix == "" || len(req.Ports) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "domain, path_prefix, and ports are required")
		return
	}

	svc, err := sm.UpdateService(id, &req)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, svc)
}

func extractServiceID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "services" {
		return strings.Join(parts[2:], "/")
	}
	return ""
}
