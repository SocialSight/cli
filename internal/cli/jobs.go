package cli

import (
	"context"
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/SocialSight/cli/internal/client"
)

// defaultWaitPollInterval and defaultWaitTimeout are the --wait-interval and
// --wait-timeout flag defaults, shared by `jobs wait` and `generate --wait`.
const (
	defaultWaitPollInterval = 3 * time.Second
	defaultWaitTimeout      = 10 * time.Minute
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
			return outputJob(cmd, job)
		},
	}
}

func newJobsWaitCmd() *cobra.Command {
	var interval, timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <job_id>",
		Short: "Poll a job until it completes or fails",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			jobID := args[0]
			stop := startSpinner(cmd, fmt.Sprintf("waiting for job %s", jobID))
			job, err := waitForJob(ctx, c, jobID, interval)
			stop()
			if err != nil {
				return err
			}

			if err := outputJob(cmd, job); err != nil {
				return err
			}
			if job.Status == client.Error {
				return errors.New("job failed")
			}
			return nil
		},
	}
	registerWaitTimingFlags(cmd, &interval, &timeout)
	return cmd
}

func registerWaitTimingFlags(cmd *cobra.Command, interval, timeout *time.Duration) {
	cmd.Flags().DurationVar(interval, "wait-interval", defaultWaitPollInterval, "how often to poll while waiting")
	cmd.Flags().DurationVar(timeout, "wait-timeout", defaultWaitTimeout, "how long to wait before giving up")
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

func waitForJob(ctx context.Context, c *client.ClientWithResponses, jobID string, interval time.Duration) (*client.JobRecord, error) {
	lastStatus := client.JobStatus("unknown")
	for {
		job, err := getJob(ctx, c, jobID)
		if err != nil {
			// The deadline can also expire mid-request, in which case the
			// HTTP client itself returns a raw "context deadline exceeded"
			// error rather than this loop's own select below ever firing.
			if ctx.Err() != nil {
				return nil, fmt.Errorf("timed out waiting for job %s (last status: %s)", jobID, lastStatus)
			}
			return nil, err
		}
		lastStatus = job.Status
		if job.Status == client.Completed || job.Status == client.Error {
			return job, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for job %s (last status: %s)", jobID, job.Status)
		case <-time.After(interval):
		}
	}
}

func outputJob(cmd *cobra.Command, job *client.JobRecord) error {
	if wantsJSON(cmd) {
		return printJSON(cmd, job)
	}
	printJob(cmd, job)
	return nil
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
