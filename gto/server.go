package gto

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Server exposes the compatible /gto/decision and /gto/decision_v2 endpoints.
type Server struct {
	service *Service
	apiKey  string
	logger  *log.Logger
}

func NewServer(service *Service, apiKey string, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{service: service, apiKey: apiKey, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/gto/decision", s.handleDecision)
	mux.HandleFunc("/gto/decision_v2", s.handleDecision)
	mux.HandleFunc("/healthz", s.handleHealth)
	return s.recover(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.service.Status())
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "invalid api key")
		return
	}
	start := time.Now()
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	var req TexasGtoDecisionReq
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := ValidateRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := s.service.Decide(r.Context(), &req)
	if err != nil {
		s.logger.Printf("[gto] decide error path=%s err=%v", r.URL.Path, err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.logger.Printf("[gto] decision path=%s position=%s stage=%s action=%s size=%d cost=%s detail=%q",
		r.URL.Path, req.CurrentPlayerPosition, req.GameStage, resp.Action, resp.RaiseSize,
		time.Since(start).String(), resp.Detail)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) authorized(r *http.Request) bool {
	if s.apiKey == "" {
		return true
	}
	got := r.Header.Get("X-API-Key")
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.apiKey)) == 1
}

func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Printf("[gto] panic path=%s err=%v", r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
