package cmd

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tillkuhn/billy-idle/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"

	"github.com/spf13/cobra"
)

// wspCmd represents the wsp command
var wspCmd = &cobra.Command{
	Use:   "wsp",
	Short: "What's up?",
	Long:  `Returns status info from the current tracker instance`,
	Run: func(cmd *cobra.Command, args []string) {
		status(cmd.Context())

	},
}

func status(ctx context.Context) {
	addr := "localhost:50051"
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) { _ = conn.Close() }(conn)
	c := pb.NewBillyClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Status(ctx, &empty.Empty{})
	if err != nil {
		log.Fatalf("could not get status: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}

func init() {
	rootCmd.AddCommand(wspCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wspCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wspCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
