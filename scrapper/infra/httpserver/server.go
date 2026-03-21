package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

type Server struct {
	service *repository.Service
}

func New(service *repository.Service) *Server {
	return &Server{service: service}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tg-chat/", s.handleChat)
	mux.HandleFunc("/links", s.handleLinks)
	return mux
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/tg-chat/")
	chatID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleRegisterChat(w, chatID)
	case http.MethodDelete:
		s.handleDeleteChat(w, chatID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRegisterChat(w http.ResponseWriter, chatID int64) {
	err := s.service.RegisterChat(chatID)
	if err != nil {
		if errors.Is(err, repository.ErrChatExists) {
			writeError(w, http.StatusConflict, "chat already exists")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteChat(w http.ResponseWriter, chatID int64) {
	err := s.service.DeleteChat(chatID)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			writeError(w, http.StatusNotFound, "chat does not exist")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLinks(w http.ResponseWriter, r *http.Request) {
	chatID, err := parseChatIDHeader(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListLinks(w, chatID)
	case http.MethodPost:
		s.handleAddLink(w, r, chatID)
	case http.MethodDelete:
		s.handleRemoveLink(w, r, chatID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListLinks(w http.ResponseWriter, chatID int64) {
	resp, listErr := s.service.ListLinks(chatID)
	if listErr != nil {
		if errors.Is(listErr, repository.ErrChatNotFound) {
			writeError(w, http.StatusNotFound, "chat does not exist")
			return
		}
		writeError(w, http.StatusBadRequest, listErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAddLink(w http.ResponseWriter, r *http.Request, chatID int64) {
	var req api.AddLinkRequest
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid add link request")
		return
	}

	resp, addErr := s.service.AddLink(chatID, req)
	if addErr != nil {
		switch {
		case errors.Is(addErr, repository.ErrChatNotFound):
			writeError(w, http.StatusNotFound, "chat does not exist")
		case errors.Is(addErr, repository.ErrLinkExists):
			writeError(w, http.StatusConflict, "link already tracked")
		default:
			writeError(w, http.StatusBadRequest, addErr.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRemoveLink(w http.ResponseWriter, r *http.Request, chatID int64) {
	var req api.RemoveLinkRequest
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid remove link request")
		return
	}

	resp, removeErr := s.service.RemoveLink(chatID, req)
	if removeErr != nil {
		if errors.Is(removeErr, repository.ErrChatNotFound) || errors.Is(removeErr, repository.ErrLinkNotFound) {
			writeError(w, http.StatusNotFound, "chat does not exist or link not found")
			return
		}
		writeError(w, http.StatusBadRequest, removeErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseChatIDHeader(r *http.Request) (int64, error) {
	rawID := strings.TrimSpace(r.Header.Get("Tg-Chat-Id"))
	if rawID == "" {
		return 0, errors.New("missing Tg-Chat-Id header")
	}
	chatID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid Tg-Chat-Id header")
	}
	return chatID, nil
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("Failed to write JSON response", slog.String("error", err.Error()))
	}
}

func writeError(w http.ResponseWriter, code int, description string) {
	writeJSON(w, code, api.ErrorResponse{
		Description: description,
		Code:        strconv.Itoa(code),
	})
}
