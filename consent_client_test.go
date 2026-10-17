package mediaconsent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGrantRequestBoundary(t *testing.T) {
	var method, authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{},"metadata":{}}`))
	}))
	defer server.Close()

	client := NewConsentClient("test-key")
	client.BaseURL = server.URL
	if err := client.Grant(context.Background(), "listener-42", DeliveryCategory, "grant-asset-8"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if authorization != "Bearer test-key" {
		t.Fatalf("authorization header was not set")
	}
}
