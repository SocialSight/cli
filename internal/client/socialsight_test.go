package client

import "testing"

func TestMCPBaseURLDerivation(t *testing.T) {
	cases := []struct {
		name       string
		apiBaseURL string
		mcpEnv     string
		want       string
		wantErr    bool
	}{
		{name: "prod default", apiBaseURL: "", want: DefaultMCPBaseURL},
		{name: "staging api", apiBaseURL: stagingAPIBaseURL, want: stagingMCPBaseURL},
		{name: "explicit env override wins", apiBaseURL: "", mcpEnv: "https://custom-mcp.test", want: "https://custom-mcp.test"},
		{name: "explicit env override wins over staging api too", apiBaseURL: stagingAPIBaseURL, mcpEnv: "https://custom-mcp.test", want: "https://custom-mcp.test"},
		{name: "unmappable custom api url errors", apiBaseURL: "http://127.0.0.1:8080", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envBaseURL, tc.apiBaseURL)
			t.Setenv(envMCPBaseURL, tc.mcpEnv)

			got, err := MCPBaseURL()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("MCPBaseURL: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
