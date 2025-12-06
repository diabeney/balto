package health

import (
	"net/http"

	"github.com/diabeney/balto/internal/api/core"
	"github.com/diabeney/balto/pkg/utils"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sm := core.GetServiceManager()
	if sm == nil {
		utils.WriteError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	stats := sm.GetHealthStats()
	utils.WriteJSON(w, http.StatusOK, stats)
}
