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
