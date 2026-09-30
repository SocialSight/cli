package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SocialSight/cli/internal/client"
)

// The backend's declared 422 schema is an array of pydantic errors, but a
// manually raised HTTPException(422, detail="...") serializes detail as a
// plain string instead. Both must decode without error.
func TestValidationErrorStringDetail(t *testing.T) {
	raw := json.RawMessage(`"no image price for model=X resolution='512p'"`)
	err := validationError(&client.HTTPValidationError{Detail: &raw})
	if err == nil || !strings.Contains(err.Error(), "no image price for model=X") {
		t.Fatalf("got %v, want the plain string surfaced verbatim", err)
	}
}

func TestValidationErrorStructuredDetail(t *testing.T) {
	raw := json.RawMessage(`[{"loc": ["body", "prompt"], "msg": "field required", "type": "missing"}]`)
	err := validationError(&client.HTTPValidationError{Detail: &raw})
	if err == nil || !strings.Contains(err.Error(), "body.prompt") || !strings.Contains(err.Error(), "field required") {
		t.Fatalf("got %v, want loc/msg rendered", err)
	}
}

func TestValidationErrorUnrecognizedShapeFallsBackToRaw(t *testing.T) {
	raw := json.RawMessage(`{"unexpected": "shape"}`)
	err := validationError(&client.HTTPValidationError{Detail: &raw})
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("got %v, want raw body surfaced as a fallback", err)
	}
}

func TestValidationErrorNilDetail(t *testing.T) {
	if err := validationError(&client.HTTPValidationError{}); err == nil {
		t.Fatal("expected a generic error for nil Detail")
	}
	if err := validationError(nil); err == nil {
		t.Fatal("expected a generic error for nil HTTPValidationError")
	}
}

// A 403 PLAN_REQUIRED means the plan doesn't include the model, not an expired
// session, so it must surface the API message and the upgrade link.
func TestPlanRequiredErrorExplainsThePlanGate(t *testing.T) {
	body := []byte(`{"detail": {"code": "PLAN_REQUIRED", "message": "Seedance 2.5 is available on paid plans only.", "model_id": "SEEDANCE_2_5", "reason": "subscription"}}`)
	err := planRequiredError(403, body)
	if err == nil {
		t.Fatal("expected a plan-required error")
	}
	for _, want := range []string{"Seedance 2.5 is available on paid plans only.", "/apps/subscribe", "socialsight model list"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("got %q, want it to contain %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "auth login") {
		t.Fatalf("got %q, must not ask the user to log in again", err)
	}
}

func TestPlanRequiredErrorIgnoresOtherResponses(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"plain 403", 403, `{"detail": "Forbidden"}`},
		{"other code", 403, `{"detail": {"code": "SOMETHING_ELSE", "message": "x"}}`},
		{"not json", 403, `nope`},
		{"other status", 429, `{"detail": {"code": "PLAN_REQUIRED", "message": "x"}}`},
	}
	for _, tc := range cases {
		if err := planRequiredError(tc.status, []byte(tc.body)); err != nil {
			t.Errorf("%s: got %v, want nil", tc.name, err)
		}
	}
	// A plain 403 still falls through to the re-login prompt.
	if err := authError(403); err == nil || !strings.Contains(err.Error(), "auth login") {
		t.Fatalf("got %v, want the re-login prompt", err)
	}
}
