package cli

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/SocialSight/cli/internal/client"
)

// waitPollInterval and waitTimeout are fixed for now; ENG-268 makes these
// configurable (--wait-interval/--wait-timeout) and adds a spinner.
const (
	waitPollInterval = 3 * time.Second
	waitTimeout      = 10 * time.Minute
)

func newJobsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Check on generation jobs",
	}
	cmd.AddCommand(newJobsGetCmd())
	cmd.AddCommand(newJobsWaitCmd())
	return cmd
}

func newJobsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <job_id>",
		Short: "Fetch the current status of a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			job, err := getJob(ctx, c, args[0])
			if err != nil {
				return err
			}
			printJob(cmd, job)
			return nil
		},
	}
}

func newJobsWaitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wait <job_id>",
		Short: "Poll a job until it completes or fails",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), waitTimeout)
			defer cancel()

			job, err := waitForJob(ctx, c, args[0])
			if err != nil {
				return err
			}
			printJob(cmd, job)
			if job.Status == client.Error {
				return fmt.Errorf("job failed")
			}
			return nil
		},
	}
}

func getJob(ctx context.Context, c *client.ClientWithResponses, jobID string) (*client.JobRecord, error) {
	resp, err := c.GetJobV1JobsJobIdGetWithResponse(ctx, jobID, nil)
	if err != nil {
		return nil, err
	}
	if resp.JSON422 != nil {
		return nil, validationError(resp.JSON422)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response (%s): %s", resp.Status(), string(resp.Body))
	}
	return resp.JSON200, nil
}

func waitForJob(ctx context.Context, c *client.ClientWithResponses, jobID string) (*client.JobRecord, error) {
	for {
		job, err := getJob(ctx, c, jobID)
		if err != nil {
			return nil, err
		}
		if job.Status == client.Completed || job.Status == client.Error {
			return job, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for job %s (last status: %s)", jobID, job.Status)
		case <-time.After(waitPollInterval):
		}
	}
}

func printJob(cmd *cobra.Command, job *client.JobRecord) {
	out := cmd.OutOrStdout()

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Job ID:\t%s\n", job.JobId)
	fmt.Fprintf(w, "Type:\t%s\n", job.JobType)
	fmt.Fprintf(w, "Status:\t%s\n", job.Status)
	fmt.Fprintf(w, "Created:\t%s\n", job.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "Updated:\t%s\n", job.LastUpdatedAt.Format(time.RFC3339))
	if job.ErrorMessage != nil {
		fmt.Fprintf(w, "Error:\t%s\n", *job.ErrorMessage)
	}
	_ = w.Flush()

	if job.Outputs != nil {
		for i, o := range *job.Outputs {
			fmt.Fprintf(out, "Output %d: %s\n", i+1, o.Url)
		}
	}
}
