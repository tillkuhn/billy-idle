package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/spf13/cobra"
)

// wspCmd represents the wsp command
var wspCmd = &cobra.Command{
	Use:   "wsp",
	Short: "What's up?",
	Long:  `Returns status info from the current tracker instance`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return status(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(wspCmd)
	// wspCmd.PersistentFlags().IntVar(&gRPCPort, "port", 50051, "Port for gRPC Communication")
}

func status(ctx context.Context) error {
	cl, cf, err := initClient()
	if err != nil {
		return err
	}
	defer cf()
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// https://github.com/grpc/grpc-go/blob/master/examples/features/wait_for_ready/main.go#L93
	r, err := cl.WhatsUp(ctx, &empty.Empty{}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(rootCmd.OutOrStdout(), "Response: %s\n", r.GetMessage())
	return nil
}
