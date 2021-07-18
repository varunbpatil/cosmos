package http

import (
	"cosmos"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) registerEndpointRoutes(r *mux.Router) {
	r.HandleFunc("/endpoints", s.findEndpoints).Methods("GET")
	r.HandleFunc("/endpoints", s.createEndpoint).Methods("POST")
	r.HandleFunc("/endpoints/{id}", s.updateEndpoint).Methods("PATCH")
	r.HandleFunc("/endpoints/{id}", s.deleteEndpoint).Methods("DELETE")

	r.HandleFunc("/endpoints/{id}/edit-form", s.editEndpointForm).Methods("GET")
	r.HandleFunc("/endpoints/{id}/rediscover", s.rediscoverEndpoint).Methods("POST")
	r.HandleFunc("/endpoints/{srcID}/{dstID}/catalog-form", s.catalogForm).Methods("GET")
}

func (s *Server) findEndpoints(w http.ResponseWriter, r *http.Request) {
	// Get the "type" if any from the query params.
	filter := cosmos.EndpointFilter{}
	if endpointType, ok := r.URL.Query()["type"]; ok {
		filter.Type = &endpointType[0]
	}

	endpoints, totalEndpoints, err := s.App.FindEndpoints(r.Context(), filter)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	ret := map[string]interface{}{
		"endpoints":      endpoints,
		"totalEndpoints": totalEndpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&ret); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) createEndpoint(w http.ResponseWriter, r *http.Request) {
	var endpoint cosmos.Endpoint
	if err := json.NewDecoder(r.Body).Decode(&endpoint); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
		return
	}

	err := s.App.CreateEndpoint(r.Context(), &endpoint)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&endpoint); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) updateEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid endpoint ID"))
		return
	}

	upd := &cosmos.EndpointUpdate{}
	if err := json.NewDecoder(r.Body).Decode(upd); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
		return
	}

	endpoint, err := s.App.UpdateEndpoint(r.Context(), id, upd)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(endpoint); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) deleteEndpoint(w http.ResponseWriter, r *http.Request) {
	// Parse endpoint id from path params.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid endpoint ID"))
		return
	}

	if err := s.App.DeleteEndpoint(r.Context(), id); err != nil {
		s.ReplyWithSanitizedError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func (s *Server) editEndpointForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid endpoint ID"))
		return
	}

	endpoint, err := s.App.FindEndpointByID(r.Context(), id)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	baseForm := s.App.MessageToForm(r.Context(), &endpoint.Connector.Spec, nil)
	baseForm.Merge(&endpoint.Config)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(baseForm); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) rediscoverEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid endpoint ID"))
		return
	}

	if err := s.App.RediscoverEndpoint(r.Context(), id); err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}
}

func (s *Server) catalogForm(w http.ResponseWriter, r *http.Request) {
	srcID, err := strconv.Atoi(mux.Vars(r)["srcID"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid source endpoint ID"))
		return
	}
	dstID, err := strconv.Atoi(mux.Vars(r)["dstID"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid destination endpoint ID"))
		return
	}

	srcEndpoint, err := s.App.FindEndpointByID(r.Context(), srcID)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}
	dstEndpoint, err := s.App.FindEndpointByID(r.Context(), dstID)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	createForm := s.App.MessageToForm(
		r.Context(),
		&srcEndpoint.Catalog,
		dstEndpoint.Connector.Spec.Spec.SupportedDestinationSyncModes,
	)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(createForm); err != nil {
		s.LogError(r, err)
	}
}
