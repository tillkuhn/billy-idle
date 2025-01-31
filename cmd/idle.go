package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/tillkuhn/billy-idle/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/spf13/cobra"
)

// idleCmd represents the idle command
var idleCmd = &cobra.Command{
	Use:   "idle",
	Short: "Suspends tracking and sets state to 'idle' for the given duration",
	Long:  `See short`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return idle(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(idleCmd)
}

func idle(ctx context.Context) error {
	cl, cf, err := initClient()
	if err != nil {
		return err
	}
	defer cf()
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// https://github.com/grpc/grpc-go/blob/master/examples/features/wait_for_ready/main.go#L93
	r, err := cl.Suspend(ctx, &pb.SuspendRequest{
		State:    pb.State_IDLE,
		Duration: durationpb.New(5 * time.Minute),
	}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(rootCmd.OutOrStdout(), "Response: tracking suspended, will be idle until %s\n", r.Until.String())
	return nil
}
