package scrapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type HTTPServer struct {
	service *scrapper.Service
}

func NewHTTPServer(service *scrapper.Service) *HTTPServer {
	return &HTTPServer{service: service}
}

func (s *HTTPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tg-chat/", s.handleChat)
	mux.HandleFunc("/links", s.handleLinks)
	return mux
}

func (s *HTTPServer) handleChat(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/tg-chat/")
	chatID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	switch r.Method {
	case http.MethodPost:
		err = s.service.RegisterChat(chatID)
		if err != nil {
			if errors.Is(err, scrapper.ErrChatExists) {
				writeError(w, http.StatusConflict, "chat already exists")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		err = s.service.DeleteChat(chatID)
		if err != nil {
			if errors.Is(err, scrapper.ErrChatNotFound) {
				writeError(w, http.StatusNotFound, "chat does not exist")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleLinks(w http.ResponseWriter, r *http.Request) {
	chatID, err := parseChatIDHeader(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		resp, err := s.service.ListLinks(chatID)
		if err != nil {
			if errors.Is(err, scrapper.ErrChatNotFound) {
				writeError(w, http.StatusNotFound, "chat does not exist")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		var req api.AddLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid add link request")
			return
		}
		resp, err := s.service.AddLink(chatID, req)
		if err != nil {
			switch {
			case errors.Is(err, scrapper.ErrChatNotFound):
				writeError(w, http.StatusNotFound, "chat does not exist")
			case errors.Is(err, scrapper.ErrLinkExists):
				writeError(w, http.StatusConflict, "link already tracked")
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		var req api.RemoveLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid remove link request")
			return
		}
		resp, err := s.service.RemoveLink(chatID, req)
		if err != nil {
			if errors.Is(err, scrapper.ErrChatNotFound) || errors.Is(err, scrapper.ErrLinkNotFound) {
				writeError(w, http.StatusNotFound, "chat does not exist or link not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func parseChatIDHeader(r *http.Request) (int64, error) {
	rawID := strings.TrimSpace(r.Header.Get("Tg-Chat-Id"))
	if rawID == "" {
		return 0, fmt.Errorf("missing Tg-Chat-Id header")
	}
	chatID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid Tg-Chat-Id header")
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
	writeJSON(w, code, api.ApiErrorResponse{
		Description: description,
		Code:        fmt.Sprintf("%d", code),
	})
}
