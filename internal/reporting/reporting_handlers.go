package reporting

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ReportingHandler handles HTTP requests for portfolio performance reporting
type ReportingHandler struct {
	service *ReportingService
}

func NewReportingHandler(service *ReportingService) *ReportingHandler {
	return &ReportingHandler{service: service}
}

// GetPortfolioPerformance handles requests for portfolio performance reports
func (h *ReportingHandler) GetPortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	// Get period from query params (default to "ALL")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "ALL"
	}

	report, err := h.service.GeneratePerformanceReport(portfolioID, period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
