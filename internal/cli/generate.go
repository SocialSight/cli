package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/SocialSight/cli/internal/client"
)

type imageFlags struct {
	model       string
	prompt      string
	aspectRatio string
	quality     string
	numOutputs  int
	seed        int
}

func registerImageFlags(cmd *cobra.Command, f *imageFlags) {
	cmd.Flags().StringVar(&f.model, "model", "", "model ID (see 'socialsight model list')")
	cmd.Flags().StringVar(&f.prompt, "prompt", "", "generation prompt")
	cmd.Flags().StringVar(&f.aspectRatio, "aspect-ratio", "", "aspect ratio, e.g. 16:9 (default: model default)")
	cmd.Flags().StringVar(&f.quality, "quality", "", "quality: 512p, 1K, 1.5K, 2K, or 4K (default: 1K)")
	cmd.Flags().IntVar(&f.numOutputs, "num-outputs", 0, "number of outputs (default: 1)")
	cmd.Flags().IntVar(&f.seed, "seed", 0, "random seed (default: random)")
	_ = cmd.MarkFlagRequired("model")
}

func (f imageFlags) toGenerationRequest(cmd *cobra.Command) client.ImageGenerationRequest {
	req := client.ImageGenerationRequest{ModelId: f.model, Prompt: f.prompt}
	if f.aspectRatio != "" {
		req.AspectRatio = &f.aspectRatio
	}
	if f.quality != "" {
		q := client.ImageGenerationRequestQuality(f.quality)
		req.Quality = &q
	}
	if cmd.Flags().Changed("num-outputs") {
		req.NumOutputs = &f.numOutputs
	}
	if cmd.Flags().Changed("seed") {
		req.Seed = &f.seed
	}
	return req
}

func (f imageFlags) toCostRequest(cmd *cobra.Command) client.ImageGenerationCostRequest {
	req := client.ImageGenerationCostRequest{ModelId: f.model}
	if f.prompt != "" {
		req.Prompt = &f.prompt
	}
	if f.aspectRatio != "" {
		req.AspectRatio = &f.aspectRatio
	}
	if f.quality != "" {
		q := client.ImageGenerationCostRequestQuality(f.quality)
		req.Quality = &q
	}
	if cmd.Flags().Changed("num-outputs") {
		req.NumOutputs = &f.numOutputs
	}
	if cmd.Flags().Changed("seed") {
		req.Seed = &f.seed
	}
	return req
}

type videoFlags struct {
	model       string
	prompt      string
	duration    int
	resolution  string
	aspectRatio string
	enableAudio bool
	seed        int
}

func registerVideoFlags(cmd *cobra.Command, f *videoFlags) {
	cmd.Flags().StringVar(&f.model, "model", "", "model ID (see 'socialsight model list')")
	cmd.Flags().StringVar(&f.prompt, "prompt", "", "generation prompt")
	cmd.Flags().IntVar(&f.duration, "duration", 0, "duration in seconds (default: 5)")
	cmd.Flags().StringVar(&f.resolution, "resolution", "", "resolution, e.g. 720p (default: 720p)")
	cmd.Flags().StringVar(&f.aspectRatio, "aspect-ratio", "", "aspect ratio, e.g. 16:9 (default: 16:9)")
	cmd.Flags().BoolVar(&f.enableAudio, "enable-audio", false, "generate audio alongside video")
	cmd.Flags().IntVar(&f.seed, "seed", 0, "random seed (default: random)")
	_ = cmd.MarkFlagRequired("model")
}

func videoDuration(seconds int) (client.VideoGenerationRequest_Duration, error) {
	var d client.VideoGenerationRequest_Duration
	err := d.FromVideoGenerationRequestDuration0(seconds)
	return d, err
}

func videoCostDuration(seconds int) (client.VideoGenerationCostRequest_Duration, error) {
	var d client.VideoGenerationCostRequest_Duration
	err := d.FromVideoGenerationCostRequestDuration0(seconds)
	return d, err
}

func (f videoFlags) toGenerationRequest(cmd *cobra.Command) (client.VideoGenerationRequest, error) {
	req := client.VideoGenerationRequest{ModelId: f.model}
	if f.prompt != "" {
		req.Prompt = &f.prompt
	}
	if f.resolution != "" {
		req.Resolution = &f.resolution
	}
	if f.aspectRatio != "" {
		req.AspectRatio = &f.aspectRatio
	}
	if cmd.Flags().Changed("enable-audio") {
		req.EnableAudio = &f.enableAudio
	}
	if cmd.Flags().Changed("seed") {
		req.Seed = &f.seed
	}
	if cmd.Flags().Changed("duration") {
		d, err := videoDuration(f.duration)
		if err != nil {
			return req, err
		}
		req.Duration = &d
	}
	return req, nil
}

func (f videoFlags) toCostRequest(cmd *cobra.Command) (client.VideoGenerationCostRequest, error) {
	req := client.VideoGenerationCostRequest{ModelId: f.model}
	if f.prompt != "" {
		req.Prompt = &f.prompt
	}
	if f.resolution != "" {
		req.Resolution = &f.resolution
	}
	if f.aspectRatio != "" {
		req.AspectRatio = &f.aspectRatio
	}
	if cmd.Flags().Changed("enable-audio") {
		req.EnableAudio = &f.enableAudio
	}
	if cmd.Flags().Changed("seed") {
		req.Seed = &f.seed
	}
	if cmd.Flags().Changed("duration") {
		d, err := videoCostDuration(f.duration)
		if err != nil {
			return req, err
		}
		req.Duration = &d
	}
	return req, nil
}

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate images or videos",
	}
	cmd.AddCommand(newGenerateImageCmd())
	cmd.AddCommand(newGenerateVideoCmd())
	cmd.AddCommand(newGenerateCostCmd())
	return cmd
}

func newGenerateImageCmd() *cobra.Command {
	var f imageFlags

	cmd := &cobra.Command{
		Use:   "image",
		Short: "Create an image generation job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if f.prompt == "" {
				return fmt.Errorf("--prompt is required")
			}
			c, err := requireClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			resp, err := c.CreateImageJobV1ImagePostWithResponse(ctx, f.toGenerationRequest(cmd))
			if err != nil {
				return err
			}
			if resp.JSON422 != nil {
				return validationError(resp.JSON422)
			}
			if resp.JSON202 == nil {
				return fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
			}
			return printJobCreated(cmd, resp.JSON202)
		},
	}
	registerImageFlags(cmd, &f)
	_ = cmd.MarkFlagRequired("prompt")
	return cmd
}

func newGenerateVideoCmd() *cobra.Command {
	var f videoFlags

	cmd := &cobra.Command{
		Use:   "video",
		Short: "Create a video generation job",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireClient()
			if err != nil {
				return err
			}

			req, err := f.toGenerationRequest(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			resp, err := c.CreateVideoJobV1VideoPostWithResponse(ctx, req)
			if err != nil {
				return err
			}
			if resp.JSON422 != nil {
				return validationError(resp.JSON422)
			}
			if resp.JSON202 == nil {
				return fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
			}
			return printJobCreated(cmd, resp.JSON202)
		},
	}
	registerVideoFlags(cmd, &f)
	return cmd
}

func newGenerateCostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Preview the credit cost of a generation before running it",
	}
	cmd.AddCommand(newGenerateCostImageCmd())
	cmd.AddCommand(newGenerateCostVideoCmd())
	return cmd
}

func newGenerateCostImageCmd() *cobra.Command {
	var f imageFlags

	cmd := &cobra.Command{
		Use:   "image",
		Short: "Preview the cost of an image generation",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			resp, err := c.GetImageGenerationCostV1GenerationImageCostPostWithResponse(ctx, f.toCostRequest(cmd))
			if err != nil {
				return err
			}
			if resp.JSON422 != nil {
				return validationError(resp.JSON422)
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Estimated cost: %d credits\n", resp.JSON200.Credits)
			return nil
		},
	}
	registerImageFlags(cmd, &f)
	return cmd
}

func newGenerateCostVideoCmd() *cobra.Command {
	var f videoFlags

	cmd := &cobra.Command{
		Use:   "video",
		Short: "Preview the cost of a video generation",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireClient()
			if err != nil {
				return err
			}

			req, err := f.toCostRequest(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			resp, err := c.GetVideoGenerationCostV1GenerationVideoCostPostWithResponse(ctx, req)
			if err != nil {
				return err
			}
			if resp.JSON422 != nil {
				return validationError(resp.JSON422)
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Estimated cost: %d credits\n", resp.JSON200.Credits)
			return nil
		},
	}
	registerVideoFlags(cmd, &f)
	return cmd
}

func printJobCreated(cmd *cobra.Command, job *client.JobCreateResponse) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Job created: %s (status: %s)\n", job.JobId, job.Status)
	fmt.Fprintf(cmd.OutOrStdout(), "Run `socialsight jobs get %s` to check on it.\n", job.JobId)
	return nil
}
