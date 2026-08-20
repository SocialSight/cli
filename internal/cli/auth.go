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

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save a SocialSight API key",
		Long: "Save a SocialSight API key so other commands can authenticate.\n" +
			"Create a key from the SocialSight dashboard, then pass it with --key\n" +
			"or paste it when prompted.",
		RunE: func(cmd *cobra.Command, args []string) error {
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
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "API key (omit to be prompted)")
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved API key",
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
		Short: "Show the currently authenticated key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, source, err := config.APIKey()
			if err != nil {
				return err
			}
			if key == "" {
				return fmt.Errorf("not logged in, run `socialsight auth login`")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			balance, err := fetchCreditBalance(ctx, client.BaseURL(), key)
			if err != nil {
				return fmt.Errorf("key from %s is not valid: %w", source, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s (from %s)\n", config.Mask(key), source)
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
