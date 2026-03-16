package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListAppHookDeliveriesSinceStopsAtCutoff(t *testing.T) {
	cutoff := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/app/hook/deliveries?per_page=100":
			w.Header().Add("Link", `<`+serverURL+`/app/hook/deliveries?per_page=100&page=2>; rel="next"`)
			writeJSONResponse(w, []AppHookDelivery{{ID: 1, Status: "ERROR", DeliveredAt: cutoff.Add(30 * time.Minute)}, {ID: 2, Status: "OK", DeliveredAt: cutoff.Add(10 * time.Minute)}})
		case "/app/hook/deliveries?per_page=100&page=2":
			writeJSONResponse(w, []AppHookDelivery{{ID: 3, Status: "ERROR", DeliveredAt: cutoff.Add(-5 * time.Minute)}, {ID: 4, Status: "ERROR", DeliveredAt: cutoff.Add(-30 * time.Minute)}})
		default:
			t.Fatalf("unexpected request %s", r.URL.RequestURI())
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(server.URL+"/", "test-token", server.Client())
	deliveries, err := client.ListAppHookDeliveriesSince(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("ListAppHookDeliveriesSince returned error: %v", err)
	}
	if len(deliveries) != 2 {
		t.Fatalf("expected 2 deliveries within cutoff, got %d", len(deliveries))
	}
	if deliveries[0].ID != 1 || deliveries[1].ID != 2 {
		t.Fatalf("unexpected delivery IDs: %+v", deliveries)
	}
}

func TestRedeliverAppHookDelivery(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/app/hook/deliveries/42/attempts" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", "test-token", server.Client())
	if err := client.RedeliverAppHookDelivery(context.Background(), 42); err != nil {
		t.Fatalf("RedeliverAppHookDelivery returned error: %v", err)
	}
	if !called {
		t.Fatal("expected redelivery endpoint to be called")
	}
}

func writeJSONResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}
