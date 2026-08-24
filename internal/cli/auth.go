package cli

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/SocialSight/cli/internal/client"
	"github.com/SocialSight/cli/internal/config"
	"github.com/SocialSight/cli/internal/oauth"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage SocialSight authentication",
	}
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthWhoamiCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var key string
	var paste bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to SocialSight",
		Long: "Signs in via your browser by default.\n" +
			"Pass --key (or --paste to be prompted) to authenticate with an API\n" +
			"key instead -- useful for CI or headless environments where a\n" +
			"browser isn't available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key != "" || paste {
				return loginWithAPIKey(cmd, key)
			}
			return loginWithBrowser(cmd)
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "API key (skips the browser flow)")
	cmd.Flags().BoolVar(&paste, "paste", false, "paste an API key interactively instead of using the browser")
	return cmd
}

func loginWithAPIKey(cmd *cobra.Command, key string) error {
	if key == "" {
		var err error
		key, err = readKey(cmd)
		if err != nil {
			return err
		}
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("no API key provided")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	balance, err := fetchCreditBalance(ctx, client.BaseURL(), key)
	if err != nil {
		return fmt.Errorf("could not verify key: %w", err)
	}

	if err := config.SaveAPIKey(key); err != nil {
		return fmt.Errorf("saving key: %w", err)
	}

	path, _ := config.Path()
	fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s. Saved to %s.\n", config.Mask(key), path)
	fmt.Fprintf(cmd.OutOrStdout(), "Credits remaining: %d\n", balance.TotalCredits)
	return nil
}

func loginWithBrowser(cmd *cobra.Command) error {
	mcpBaseURL, err := client.MCPBaseURL()
	if err != nil {
		return fmt.Errorf("%w (or run `socialsight auth login --paste` instead)", err)
	}

	tok, err := oauth.Login(cmd.Context(), oauth.LoginOptions{
		MCPBaseURL: mcpBaseURL,
		Output:     cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("browser login failed: %w (you can also run `socialsight auth login --paste`)", err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	balance, err := fetchCreditBalance(ctx, client.BaseURL(), tok.AccessToken)
	if err != nil {
		return fmt.Errorf("signed in, but couldn't verify the session: %w", err)
	}

	if err := config.SaveOAuth(tok.AccessToken, tok.RefreshToken, tok.ExpiresAt, mcpBaseURL); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Signed in to SocialSight.")
	fmt.Fprintf(cmd.OutOrStdout(), "Credits remaining: %d\n", balance.TotalCredits)
	return nil
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Sign out (forget the saved credential)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.DeleteAPIKey(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			if os.Getenv("SOCIALSIGHT_API_KEY") != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Note: SOCIALSIGHT_API_KEY is still set in your environment and will still be used.")
			}
			return nil
		},
	}
}

func newAuthWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cred, err := config.Load()
			if err != nil {
				return err
			}
			if cred.Token == "" {
				return fmt.Errorf("not logged in, run `socialsight auth login`")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			balance, err := fetchCreditBalance(ctx, client.BaseURL(), cred.Token)
			if err != nil {
				return fmt.Errorf("session from %s is not valid: %w", cred.Source, err)
			}

			if cred.Method == config.MethodOAuth {
				fmt.Fprintf(cmd.OutOrStdout(), "Signed in via browser login (from %s)\n", cred.Source)
				if !cred.ExpiresAt.IsZero() {
					fmt.Fprintf(cmd.OutOrStdout(), "Session expires: %s\n", cred.ExpiresAt.Format(time.RFC3339))
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s (from %s)\n", config.Mask(cred.Token), cred.Source)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Credits remaining: %d\n", balance.TotalCredits)
			return nil
		},
	}
}

// readKey prompts for an API key, masking input on a terminal and falling
// back to a plain line read when stdin isn't a TTY (e.g. piped input).
func readKey(cmd *cobra.Command) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Enter your SocialSight API key: ")

	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}

// fetchCreditBalance validates apiKey against the backend and doubles as the
// CLI's "whoami" probe: there's no dedicated public identity endpoint, but
// GET /v1/credits requires a provisioned principal, so a successful response
// both confirms the key works and returns something useful to show.
func fetchCreditBalance(ctx context.Context, baseURL, apiKey string) (*client.CreditBalanceResponse, error) {
	c, err := client.NewAuthenticated(baseURL, apiKey)
	if err != nil {
		return nil, err
	}

	resp, err := c.GetCreditsV1CreditsGetWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
	}
	return resp.JSON200, nil
}
