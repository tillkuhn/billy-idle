package cmd

import (
	"context"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tillkuhn/billy-idle/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/spf13/cobra"
)

var (
	gRPCPort int
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

func status(ctx context.Context) error {
	addr := "localhost:" + strconv.Itoa(gRPCPort)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) { _ = conn.Close() }(conn)
	c := pb.NewBillyClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// https://github.com/grpc/grpc-go/blob/master/examples/features/wait_for_ready/main.go#L93
	r, err := c.Status(ctx, &empty.Empty{}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}
	_, _ = rootCmd.OutOrStdout().Write([]byte("Response: " + r.GetMessage() + "\n"))
	// log.Printf("Greeting: %s", r.GetMessage())
	return nil
}

func init() {
	rootCmd.AddCommand(wspCmd)
	wspCmd.PersistentFlags().IntVar(&gRPCPort, "port", 50051, "Port for gRPC Communication")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wspCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wspCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
