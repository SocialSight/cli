package cli

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/SocialSight/cli/internal/client"
)

func newModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Browse available generation models",
	}
	cmd.AddCommand(newModelListCmd())
	cmd.AddCommand(newModelInfoCmd())
	return cmd
}

func newModelListCmd() *cobra.Command {
	var modelType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, args []string) error {
			models, err := fetchModels(cmd, modelType)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tMODALITY\tNAME")
			for _, m := range models {
				fmt.Fprintf(w, "%s\t%s\t%s\n", stringField(m, "id"), stringField(m, "modality"), modelDisplayName(m))
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&modelType, "type", "all", "filter by type: image, video, or all")
	return cmd
}

func newModelInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <model_id>",
		Short: "Show details for one model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			models, err := fetchModels(cmd, "all")
			if err != nil {
				return err
			}
			for _, m := range models {
				if stringField(m, "id") == args[0] {
					printModelInfo(cmd, m)
					return nil
				}
			}
			return fmt.Errorf("model %q not found, run `socialsight model list`", args[0])
		},
	}
}

func fetchModels(cmd *cobra.Command, modelType string) ([]map[string]interface{}, error) {
	t := client.GenerationModelType(modelType)
	if !t.Valid() {
		return nil, fmt.Errorf("invalid --type %q, must be image, video, or all", modelType)
	}

	c, err := requireClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	resp, err := c.GetGenerationModelsV1ModelsGenerationGetWithResponse(ctx, &client.GetGenerationModelsV1ModelsGenerationGetParams{Type: &t})
	if err != nil {
		return nil, err
	}
	if resp.JSON422 != nil {
		return nil, validationError(resp.JSON422)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
	}
	return resp.JSON200.Models, nil
}

func printModelInfo(cmd *cobra.Command, m map[string]interface{}) {
	out := cmd.OutOrStdout()
	explore, _ := m["explore"].(map[string]interface{})

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	row := func(label, value string) {
		if value == "" {
			return
		}
		fmt.Fprintf(w, "%s:\t%s\n", label, value)
	}

	row("ID", stringField(m, "id"))
	row("Modality", stringField(m, "modality"))
	row("Name", stringField(explore, "display_name"))
	row("Description", stringField(explore, "description"))
	row("Recommendation", stringField(explore, "recommendation"))
	row("Strengths", strings.Join(stringSliceField(explore, "strengths"), ", "))
	row("Cautions", strings.Join(stringSliceField(explore, "cautions"), ", "))
	row("Aspect ratios", strings.Join(stringSliceField(m, "supported_aspect_ratios"), ", "))
	row("Resolutions", strings.Join(stringSliceField(m, "supported_resolutions"), ", "))
	row("Durations", strings.Join(stringSliceField(m, "supported_durations"), ", "))
	if v, ok := m["supports_audio"].(bool); ok {
		row("Supports audio", fmt.Sprintf("%v", v))
	}
	_ = w.Flush()
}

func modelDisplayName(m map[string]interface{}) string {
	if explore, ok := m["explore"].(map[string]interface{}); ok {
		if name := stringField(explore, "display_name"); name != "" {
			return name
		}
	}
	return stringField(m, "id")
}

func stringField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func stringSliceField(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
