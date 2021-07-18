package http

import (
	"cosmos"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) registerSyncRoutes(r *mux.Router) {
	r.HandleFunc("/syncs", s.findSyncs).Methods("GET")
	r.HandleFunc("/syncs/{id}", s.findSyncs).Methods("GET")
	r.HandleFunc("/syncs", s.createSync).Methods("POST")
	r.HandleFunc("/syncs/{id}", s.updateSync).Methods("PATCH")
	r.HandleFunc("/syncs/{id}", s.deleteSync).Methods("DELETE")

	r.HandleFunc("/syncs/{id}/edit-form", s.editSyncForm).Methods("GET")
	r.HandleFunc("/syncs/{id}/sync-now", s.syncNow).Methods("POST")
}

func (s *Server) findSyncs(w http.ResponseWriter, r *http.Request) {
	filter := cosmos.SyncFilter{}

	if v, ok := mux.Vars(r)["id"]; ok {
		syncID, err := strconv.Atoi(v)
		if err != nil {
			s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid sync ID"))
			return
		}
		filter.ID = &syncID
	}

	syncs, totalSyncs, err := s.App.FindSyncs(r.Context(), filter)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	ret := map[string]interface{}{
		"syncs":      syncs,
		"totalSyncs": totalSyncs,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&ret); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) createSync(w http.ResponseWriter, r *http.Request) {
	var sync cosmos.Sync
	if err := json.NewDecoder(r.Body).Decode(&sync); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
		return
	}

	err := s.App.CreateSync(r.Context(), &sync)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&sync); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) updateSync(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid sync ID"))
		return
	}

	upd := &cosmos.SyncUpdate{}
	if err = json.NewDecoder(r.Body).Decode(upd); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
		return
	}

	sync, err := s.App.UpdateSync(r.Context(), id, upd)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sync); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) deleteSync(w http.ResponseWriter, r *http.Request) {
	// Parse sync id from path params.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid sync ID"))
		return
	}

	if err := s.App.DeleteSync(r.Context(), id); err != nil {
		s.ReplyWithSanitizedError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func (s *Server) editSyncForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid sync ID"))
		return
	}

	sync, err := s.App.FindSyncByID(r.Context(), id)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	baseForm := s.App.MessageToForm(
		r.Context(),
		&sync.SourceEndpoint.Catalog,
		sync.DestinationEndpoint.Connector.Spec.Spec.SupportedDestinationSyncModes,
	)
	baseForm.Merge(&sync.Config)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(baseForm); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) syncNow(w http.ResponseWriter, r *http.Request) {
	syncID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid sync ID"))
		return
	}

	runOptions := &cosmos.RunOptions{}
	if err := json.NewDecoder(r.Body).Decode(runOptions); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid run options"))
		return
	}

	if err := s.App.Schedule(&syncID, runOptions); err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}
}
