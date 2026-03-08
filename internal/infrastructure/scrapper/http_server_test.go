package scrapper

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appscrapper "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/scrapper"
)

func TestHTTPServer_ChatAndLinksLifecycle(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register chat status = %d, expected 200", rec.Code)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo","tags":["work"]}`))
	addReq.Header.Set("Tg-Chat-Id", "1")
	addRec := httptest.NewRecorder()
	handler.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add link status = %d, expected 200", addRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/links", nil)
	getReq.Header.Set("Tg-Chat-Id", "1")
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get links status = %d, expected 200", getRec.Code)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	delReq.Header.Set("Tg-Chat-Id", "1")
	delRec := httptest.NewRecorder()
	handler.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete link status = %d, expected 200", delRec.Code)
	}

	finalGetReq := httptest.NewRequest(http.MethodGet, "/links", nil)
	finalGetReq.Header.Set("Tg-Chat-Id", "1")
	finalGetRec := httptest.NewRecorder()
	handler.ServeHTTP(finalGetRec, finalGetReq)
	if finalGetRec.Code != http.StatusOK {
		t.Fatalf("final get links status = %d, expected 200", finalGetRec.Code)
	}
	var finalPayload struct {
		Links []any `json:"links"`
		Size  int   `json:"size"`
	}
	if err := json.NewDecoder(finalGetRec.Body).Decode(&finalPayload); err != nil {
		t.Fatalf("decode final list response: %v", err)
	}
	if finalPayload.Size != 0 {
		t.Fatalf("expected no links after delete, got %d", finalPayload.Size)
	}
}

func TestHTTPServer_DeleteFromUnknownChatDoesNotAffectExistingLinks(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	registerReq := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	registerRec := httptest.NewRecorder()
	handler.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register chat status = %d, expected 200", registerRec.Code)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	addReq.Header.Set("Tg-Chat-Id", "1")
	addRec := httptest.NewRecorder()
	handler.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add link status = %d, expected 200", addRec.Code)
	}

	deleteUnknownReq := httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	deleteUnknownReq.Header.Set("Tg-Chat-Id", "999")
	deleteUnknownRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteUnknownRec, deleteUnknownReq)
	if deleteUnknownRec.Code == http.StatusOK {
		t.Fatalf("expected non-200 for delete in unknown chat, got %d", deleteUnknownRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/links", nil)
	getReq.Header.Set("Tg-Chat-Id", "1")
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get links status = %d, expected 200", getRec.Code)
	}

	var payload struct {
		Links []any `json:"links"`
		Size  int   `json:"size"`
	}
	if err := json.NewDecoder(getRec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if payload.Size != 1 {
		t.Fatalf("expected links to remain unchanged, size=1 got %d", payload.Size)
	}
}

func TestHTTPServer_AddLinkUnknownChat(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	req.Header.Set("Tg-Chat-Id", "2")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("add link for unknown chat status = %d, expected non-200", rec.Code)
	}
}

func TestHTTPServer_AddLinkAfterChatDeletion(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create chat status = %d, expected 200", createRec.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/tg-chat/1", nil)
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete chat status = %d, expected 200", deleteRec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	req.Header.Set("Tg-Chat-Id", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected non-200 after deleting chat, got %d", rec.Code)
	}
}

func TestHTTPServer_DeleteNonExistentChat(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodDelete, "/tg-chat/1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete non-existent chat status = %d, expected 404", rec.Code)
	}
}

func TestHTTPServer_RegisterExistingChatReturnsConflict(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	first := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first register status = %d, expected 200", firstRec.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusConflict {
		t.Fatalf("second register status = %d, expected 409", secondRec.Code)
	}
}

func TestHTTPServer_AddDuplicateLinkReturnsConflict(t *testing.T) {
	server := NewHTTPServer(appscrapper.NewService())
	handler := server.Handler()

	register := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	registerRec := httptest.NewRecorder()
	handler.ServeHTTP(registerRec, register)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register status = %d, expected 200", registerRec.Code)
	}

	add1 := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	add1.Header.Set("Tg-Chat-Id", "1")
	add1Rec := httptest.NewRecorder()
	handler.ServeHTTP(add1Rec, add1)
	if add1Rec.Code != http.StatusOK {
		t.Fatalf("first add status = %d, expected 200", add1Rec.Code)
	}

	add2 := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(`{"link":"https://github.com/user/repo"}`))
	add2.Header.Set("Tg-Chat-Id", "1")
	add2Rec := httptest.NewRecorder()
	handler.ServeHTTP(add2Rec, add2)
	if add2Rec.Code != http.StatusConflict {
		t.Fatalf("second add status = %d, expected 409", add2Rec.Code)
	}
}
