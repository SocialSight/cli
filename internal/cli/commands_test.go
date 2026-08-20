package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SocialSight/cli/internal/config"
)

const testAPIKey = "ss_live_commandstestcommandstest"

// setupAuthedCLI logs in against a fake server and returns it for the
// caller to attach further handlers to.
func setupAuthedCLI(t *testing.T, extra http.HandlerFunc) *httptest.Server {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/credits", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total_credits": 100, "subscription_credits": 100, "other_credits": 0}`))
	})
	if extra != nil {
		mux.HandleFunc("/", extra)
	}

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("SOCIALSIGHT_API_BASE_URL", srv.URL)

	if err := config.SaveAPIKey(testAPIKey); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	return srv
}

func requireAuthHeader(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
		t.Fatalf("missing/wrong Authorization header on %s", r.URL.Path)
	}
}

func TestModelListAndInfo(t *testing.T) {
	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		if r.URL.Path != "/v1/models/generation" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"models": [
			{"id": "GEMINI_3_IMAGE", "modality": "image", "supported_aspect_ratios": ["1:1", "16:9"],
			 "explore": {"display_name": "Gemini 3 Image Pro", "description": "High quality.", "strengths": ["quality"], "cautions": []}}
		]}`))
	})

	out, err := run(t, "model", "list")
	if err != nil {
		t.Fatalf("model list: %v", err)
	}
	if !strings.Contains(out, "GEMINI_3_IMAGE") || !strings.Contains(out, "Gemini 3 Image Pro") {
		t.Fatalf("unexpected model list output: %s", out)
	}

	out, err = run(t, "model", "info", "GEMINI_3_IMAGE")
	if err != nil {
		t.Fatalf("model info: %v", err)
	}
	if !strings.Contains(out, "High quality.") || !strings.Contains(out, "1:1, 16:9") {
		t.Fatalf("unexpected model info output: %s", out)
	}

	if _, err := run(t, "model", "info", "NO_SUCH_MODEL"); err == nil {
		t.Fatal("expected error for unknown model id")
	}
}

func TestGenerateImageSuccessAndValidationError(t *testing.T) {
	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		if r.URL.Path != "/v1/image" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Quality string `json:"quality"`
		}
		json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")
		if req.Quality == "512p" {
			// Mirrors a real observed case: manually-raised HTTPException
			// serializes `detail` as a plain string, not the pydantic array
			// shape the declared schema advertises.
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"detail": "no image price for model=X resolution='512p'"}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"job_id": "job-123", "status": "pending"}`))
	})

	out, err := run(t, "generate", "image", "--model", "GEMINI_3_IMAGE", "--prompt", "a duck", "--quality", "1K")
	if err != nil {
		t.Fatalf("generate image: %v", err)
	}
	if !strings.Contains(out, "job-123") {
		t.Fatalf("unexpected output: %s", out)
	}

	_, err = run(t, "generate", "image", "--model", "GEMINI_3_IMAGE", "--prompt", "a duck", "--quality", "512p")
	if err == nil || !strings.Contains(err.Error(), "no image price") {
		t.Fatalf("got %v, want the backend's plain-string 422 surfaced", err)
	}
}

func TestJobsGetAndWait(t *testing.T) {
	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		if r.URL.Path != "/v1/jobs/job-123" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"job_id": "job-123", "job_type": "IMAGE", "status": "completed",
			"created_at": "2026-01-01T00:00:00Z", "last_updated_at": "2026-01-01T00:00:05Z",
			"outputs": [{"mediaId": "m1", "source": "generated", "url": "https://example.com/out.png",
			             "modality": "image", "contentType": "image/png",
			             "createdAt": "2026-01-01T00:00:05Z", "updatedAt": "2026-01-01T00:00:05Z"}]
		}`))
	})

	out, err := run(t, "jobs", "get", "job-123")
	if err != nil {
		t.Fatalf("jobs get: %v", err)
	}
	if !strings.Contains(out, "completed") || !strings.Contains(out, "https://example.com/out.png") {
		t.Fatalf("unexpected jobs get output: %s", out)
	}

	// Already completed on the first poll, so `wait` returns immediately.
	out, err = run(t, "jobs", "wait", "job-123")
	if err != nil {
		t.Fatalf("jobs wait: %v", err)
	}
	if !strings.Contains(out, "completed") {
		t.Fatalf("unexpected jobs wait output: %s", out)
	}
}
