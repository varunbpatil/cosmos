package http

import (
	"cosmos"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) registerArtifactRoutes(r *mux.Router) {
	r.HandleFunc("/artifacts/{runID}/{artifactID}", s.getArtifact).Methods("GET")
}

func (s *Server) getArtifact(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.Atoi(mux.Vars(r)["runID"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid run ID"))
		return
	}
	artifactID, err := strconv.Atoi(mux.Vars(r)["artifactID"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid artifact ID"))
		return
	}

	run, err := s.App.FindRunByID(r.Context(), runID)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	artifactory, err := s.App.GetArtifactory(run.SyncID, run.ExecutionDate)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	data, err := s.App.GetArtifactData(artifactory, artifactID)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	fmt.Fprintf(w, string(data))
}
