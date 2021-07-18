package http

import (
	"cosmos"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) registerConnectorRoutes(r *mux.Router) {
	r.HandleFunc("/connectors", s.findConnectors).Methods("GET")
	r.HandleFunc("/connectors", s.createConnector).Methods("POST")
	r.HandleFunc("/connectors/{id}", s.updateConnector).Methods("PATCH")
	r.HandleFunc("/connectors/{id}", s.deleteConnector).Methods("DELETE")

	r.HandleFunc("/connectors/{id}/connection-spec-form", s.connectionSpecForm).Methods("GET")
	r.HandleFunc("/connectors/destination-types", s.getDestinationTypes).Methods("GET")
}

func (s *Server) findConnectors(w http.ResponseWriter, r *http.Request) {
	// Get the "type" if any from the query params.
	filter := cosmos.ConnectorFilter{}
	if connectorType, ok := r.URL.Query()["type"]; ok {
		filter.Type = &connectorType[0]
	}

	connectors, totalConnectors, err := s.App.FindConnectors(r.Context(), filter)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	ret := map[string]interface{}{
		"connectors":      connectors,
		"totalConnectors": totalConnectors,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&ret); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) createConnector(w http.ResponseWriter, r *http.Request) {
	var connector cosmos.Connector
	if err := json.NewDecoder(r.Body).Decode(&connector); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
		return
	}

	err := s.App.CreateConnector(r.Context(), &connector)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&connector); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) updateConnector(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid connector ID"))
		return
	}

	upd := &cosmos.ConnectorUpdate{}
	if err := json.NewDecoder(r.Body).Decode(upd); err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid JSON body"))
		return
	}

	connector, err := s.App.UpdateConnector(r.Context(), id, upd)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(connector); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) deleteConnector(w http.ResponseWriter, r *http.Request) {
	// Parse connector id from path params.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid connector ID"))
		return
	}

	if err := s.App.DeleteConnector(r.Context(), id); err != nil {
		s.ReplyWithSanitizedError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func (s *Server) connectionSpecForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.ReplyWithSanitizedError(w, r, cosmos.Errorf(cosmos.EINVALID, "Invalid connector ID"))
		return
	}

	connector, err := s.App.FindConnectorByID(r.Context(), id)
	if err != nil {
		s.ReplyWithSanitizedError(w, r, err)
		return
	}

	// If the connection spec is nil, trigger a connector update which will fetch the connection spec.
	if connector.Spec.Spec == nil {
		var err error
		connector, err = s.App.UpdateConnector(r.Context(), id, &cosmos.ConnectorUpdate{})
		if err != nil {
			s.ReplyWithSanitizedError(w, r, err)
			return
		}
	}

	createForm := s.App.MessageToForm(r.Context(), &connector.Spec, nil)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(createForm); err != nil {
		s.LogError(r, err)
	}
}

func (s *Server) getDestinationTypes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cosmos.DestinationTypes); err != nil {
		s.LogError(r, err)
	}
}
