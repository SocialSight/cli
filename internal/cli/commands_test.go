package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
		_, _ = w.Write([]byte(`{"total_credits": 100, "subscription_credits": 100, "other_credits": 0}`))
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
		_, _ = w.Write([]byte(`{"models": [
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
		_ = json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")
		if req.Quality == "512p" {
			// Mirrors a real observed case: manually-raised HTTPException
			// serializes `detail` as a plain string, not the pydantic array
			// shape the declared schema advertises.
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"detail": "no image price for model=X resolution='512p'"}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"job_id": "job-123", "status": "pending"}`))
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
		_, _ = w.Write([]byte(`{
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

func TestJobsWaitPollsUntilCompleted(t *testing.T) {
	var calls atomic.Int32

	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		if r.URL.Path != "/v1/jobs/job-poll" {
			http.NotFound(w, r)
			return
		}
		status := "completed"
		if calls.Add(1) < 3 {
			status = "processing"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"job_id": "job-poll", "job_type": "IMAGE", "status": %q,
			"created_at": "2026-01-01T00:00:00Z", "last_updated_at": "2026-01-01T00:00:00Z"
		}`, status)
	})

	out, err := run(t, "jobs", "wait", "job-poll", "--wait-interval", "5ms", "--wait-timeout", "2s")
	if err != nil {
		t.Fatalf("jobs wait: %v", err)
	}
	if !strings.Contains(out, "completed") {
		t.Fatalf("unexpected output: %s", out)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("expected 3 polls, got %d", got)
	}
}

func TestJobsWaitTimesOut(t *testing.T) {
	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"job_id": "job-stuck", "job_type": "IMAGE", "status": "processing",
			"created_at": "2026-01-01T00:00:00Z", "last_updated_at": "2026-01-01T00:00:00Z"
		}`))
	})

	_, err := run(t, "jobs", "wait", "job-stuck", "--wait-interval", "5ms", "--wait-timeout", "30ms")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("got %v, want a timeout error", err)
	}
}

func TestGenerateImageWaitPollsToCompletion(t *testing.T) {
	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/image":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"job_id": "job-wait", "status": "pending"}`))
		case "/v1/jobs/job-wait":
			_, _ = w.Write([]byte(`{
				"job_id": "job-wait", "job_type": "IMAGE", "status": "completed",
				"created_at": "2026-01-01T00:00:00Z", "last_updated_at": "2026-01-01T00:00:00Z",
				"outputs": [{"mediaId": "m1", "source": "generated", "url": "https://example.com/out.png",
				             "modality": "image", "contentType": "image/png",
				             "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}]
			}`))
		default:
			http.NotFound(w, r)
		}
	})

	out, err := run(t, "generate", "image", "--model", "M", "--prompt", "a duck", "--wait", "--wait-interval", "5ms")
	if err != nil {
		t.Fatalf("generate image --wait: %v", err)
	}
	if !strings.Contains(out, "completed") || !strings.Contains(out, "https://example.com/out.png") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestJSONOutputMode(t *testing.T) {
	setupAuthedCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/models/generation":
			_, _ = w.Write([]byte(`{"models": [{"id": "M1", "modality": "image"}]}`))
		case "/v1/generation/image/cost":
			_, _ = w.Write([]byte(`{"credits": 42, "modelId": "M1", "jobType": "IMAGE"}`))
		default:
			http.NotFound(w, r)
		}
	})

	out, err := run(t, "--json", "model", "list")
	if err != nil {
		t.Fatalf("model list --json: %v", err)
	}
	var models []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &models); err != nil {
		t.Fatalf("model list --json did not produce valid JSON: %v\noutput: %s", err, out)
	}
	if len(models) != 1 || models[0]["id"] != "M1" {
		t.Fatalf("unexpected decoded models: %+v", models)
	}

	out, err = run(t, "--json", "generate", "cost", "image", "--model", "M1")
	if err != nil {
		t.Fatalf("generate cost image --json: %v", err)
	}
	var cost map[string]interface{}
	if err := json.Unmarshal([]byte(out), &cost); err != nil {
		t.Fatalf("generate cost --json did not produce valid JSON: %v\noutput: %s", err, out)
	}
	if cost["credits"].(float64) != 42 {
		t.Fatalf("unexpected decoded cost: %+v", cost)
	}
}
