package campusnet

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTestConnectionRejectsCaptivePortalPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/online":
			w.WriteHeader(http.StatusNoContent)
		case "/portal":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("请先登录认证系统"))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer server.Close()

	client := New(time.Second, "test")
	if !client.TestConnection(server.URL+"/online", "") {
		t.Fatal("204 response should be online")
	}
	if client.TestConnection(server.URL+"/portal", "") {
		t.Fatal("Chinese captive portal page should be offline")
	}
	if !client.TestConnection(server.URL+"/plain", "") {
		t.Fatal("ordinary 200 response should be online")
	}
	if client.TestConnection(server.URL+"/generate_204", "") {
		t.Fatal("200 response from generate_204 should be offline")
	}
}
