package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/SocialSight/cli/internal/client"
	"github.com/SocialSight/cli/internal/config"
)

// requireClient builds an authenticated API client from the stored/env
// credential, or returns a clear error if the user isn't logged in.
func requireClient() (*client.ClientWithResponses, error) {
	key, _, err := config.APIKey()
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, fmt.Errorf("not logged in, run `socialsight auth login`")
	}
	return client.NewAuthenticated(client.BaseURL(), key)
}

// authError returns a friendly re-login prompt for 401/403 responses (an
// OAuth session past its 1-day expiry hits this the same way a bad API key
// would), or nil for any other status. Every authenticated command should
// check this before falling back to a generic "unexpected response" error.
func authError(statusCode int) error {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return errors.New("not authenticated (or your session expired) -- run `socialsight auth login`")
	}
	return nil
}

// subscribeURL is where a plan-gated refusal sends the user to upgrade.
const subscribeURL = "https://www.socialsight.ai/apps/subscribe?utm_source=cli&utm_medium=cli_error&utm_campaign=plan_required"

// planRequiredError explains a 403 PLAN_REQUIRED response: the user's plan
// doesn't include the requested model (e.g. Seedance or Seedream without a
// subscription, Wan 3.0 below Standard). It returns nil for any other response,
// so generation commands must check it before authError, which would otherwise
// report that 403 as an expired session.
func planRequiredError(statusCode int, body []byte) error {
	if statusCode != http.StatusForbidden {
		return nil
	}
	var payload struct {
		Detail struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Detail.Code != "PLAN_REQUIRED" {
		return nil
	}
	msg := payload.Detail.Message
	if msg == "" {
		msg = "your plan doesn't include this model"
	}
	return fmt.Errorf("%s\nPick another model (see `socialsight model list`) or upgrade your plan: %s", msg, subscribeURL)
}

// validationError renders an HTTPValidationError. The backend's declared
// schema for this field is an array of structured pydantic errors, but a
// manually raised HTTPException(422, detail="...") serializes it as a plain
// string instead (FastAPI doesn't validate error bodies against the
// documented schema) -- so this tolerates either shape, plus anything else
// by falling back to the raw JSON.
func validationError(ve *client.HTTPValidationError) error {
	if ve == nil || ve.Detail == nil {
		return errors.New("request was rejected")
	}
	raw := *ve.Detail

	var detail string
	if err := json.Unmarshal(raw, &detail); err == nil {
		return errors.New(detail)
	}

	var items []client.ValidationError
	if err := json.Unmarshal(raw, &items); err == nil {
		msg := "request was rejected:"
		for _, d := range items {
			parts := make([]string, len(d.Loc))
			for i, item := range d.Loc {
				parts[i] = locItemString(item)
			}
			msg += fmt.Sprintf("\n  %s: %s", strings.Join(parts, "."), d.Msg)
		}
		return errors.New(msg)
	}

	return fmt.Errorf("request was rejected: %s", string(raw))
}

// locItemString renders one ValidationError.loc entry (a string field name
// or an integer array index) as plain text.
func locItemString(item client.ValidationError_Loc_Item) string {
	b, err := json.Marshal(item)
	if err != nil {
		return "?"
	}
	return strings.Trim(string(b), `"`)
}
