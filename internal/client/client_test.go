package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SocialSight/cli/internal/client"
)

func TestGetGenerationModelsWithResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models/generation" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models": [{"id": "nano_banana_2", "type": "image"}]}`))
	}))
	defer srv.Close()

	c, err := client.NewClientWithResponses(srv.URL, client.WithRequestEditorFn(
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer test-key")
			return nil
		},
	))
	if err != nil {
		t.Fatalf("NewClientWithResponses: %v", err)
	}

	resp, err := c.GetGenerationModelsV1ModelsGenerationGetWithResponse(context.Background(), nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status: %d, body: %s", resp.StatusCode(), resp.Body)
	}
	if resp.JSON200 == nil || len(resp.JSON200.Models) != 1 {
		t.Fatalf("unexpected parsed body: %+v", resp.JSON200)
	}
	if resp.JSON200.Models[0]["id"] != "nano_banana_2" {
		t.Fatalf("unexpected model id: %+v", resp.JSON200.Models[0])
	}
}
