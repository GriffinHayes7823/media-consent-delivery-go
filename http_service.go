package mediaconsent

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type Service struct {
	Consent ConsentGateway
}

type consentCommand struct {
	UserID      string `json:"user_id"`
	Category    string `json:"category"`
	OperationID string `json:"operation_id"`
}

func (s Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /consent/grant", s.changeConsent(true))
	mux.HandleFunc("POST /consent/revoke", s.changeConsent(false))
	mux.HandleFunc("POST /delivery/decide", s.decide)
	return mux
}

func (s Service) changeConsent(grant bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var command consentCommand
		if err := decodeJSON(r, &command); err != nil || command.UserID == "" || command.Category == "" || command.OperationID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
			return
		}
		var err error
		state := "revoked"
		if grant {
			err = s.Consent.Grant(r.Context(), command.UserID, command.Category, command.OperationID)
			state = "granted"
		} else {
			err = s.Consent.Revoke(r.Context(), command.UserID, command.Category, command.OperationID)
		}
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"category": command.Category, "state": state})
	}
}

func (s Service) decide(w http.ResponseWriter, r *http.Request) {
	var input DeliveryRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	decision, err := DecideDelivery(r.Context(), s.Consent, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeServiceError(w http.ResponseWriter, err error) {
	var apiErr *InfraiError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Code, "message": apiErr.Message})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "consent_service_error"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
