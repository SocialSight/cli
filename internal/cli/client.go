package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/SocialSight/cli/internal/client"
	"github.com/SocialSight/cli/internal/config"
)

// requireClient builds an authenticated API client from the stored/env API
// key, or returns a clear error if the user isn't logged in.
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
