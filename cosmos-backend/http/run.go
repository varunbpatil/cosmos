package http

import (
	"cosmos"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) registerRunRoutes(r *mux.Router) {
	r.HandleFunc("/runs", s.findRuns).Methods("POST")
	r.HandleFunc("/runs/{id}", s.findRuns).Methods("GET")

	r.HandleFunc("/runs/{id}/cancel", s.cancelRun).Methods("POST")
}

func (s *Server) findRuns(w http.ResponseWriter, r *http.Request) {
	filter := cosmos.RunFilter{}

	switch r.Method {
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
			return
		}
	case http.MethodGet:
		runID, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid run ID"))
			return
		}
		filter.ID = &runID
	default:
		panic("Unhandled request method in findRuns")
	}

	runs, totalRuns, err := s.App.FindRuns(r.Context(), filter)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	ret := map[string]interface{}{
		"runs":      runs,
		"totalRuns": totalRuns,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&ret); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) cancelRun(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid run ID"))
		return
	}
	if err := s.App.CancelRun(r.Context(), runID); err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}
}
