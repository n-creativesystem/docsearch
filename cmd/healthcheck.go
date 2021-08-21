package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/n-creativesystem/docsearch/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	healthCheckCmd = &cobra.Command{
		Use:   "health",
		Short: "Health check for node",
		Long:  "Health check for node",
		RunE: func(cmd *cobra.Command, args []string) error {
			grpcAddress = viper.GetString("grpc_address")

			certificateFile = viper.GetString("certificate_file")
			commonName = viper.GetString("common_name")

			c, err := client.NewGRPCClientWithContextTLS(grpcAddress, context.Background(), certificateFile, commonName)
			if err != nil {
				return err
			}
			defer func() {
				_ = c.Close()
			}()

			lResp, err := c.LivenessCheck()
			if err != nil {
				return err
			}

			rResp, err := c.ReadinessCheck()
			if err != nil {
				return err
			}

			resp := map[string]bool{
				"liveness":   lResp.Alive,
				"readiness:": rResp.Ready,
			}

			respBytes, err := json.Marshal(resp)
			if err != nil {
				return err
			}

			fmt.Println(string(respBytes))

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(healthCheckCmd)

	cobra.OnInitialize(initialize(nil, nil))

	healthCheckCmd.PersistentFlags().StringVar(&configFile, "config-file", "", "config file. if omitted, blast.yaml in /etc and home directory will be searched")
	healthCheckCmd.PersistentFlags().StringVar(&grpcAddress, "grpc-address", ":9000", "gRPC server listen address")
	healthCheckCmd.PersistentFlags().StringVar(&certificateFile, "certificate-file", "", "path to the client server TLS certificate file")
	healthCheckCmd.PersistentFlags().StringVar(&commonName, "common-name", "", "certificate common name")

	_ = viper.BindPFlag("grpc_address", healthCheckCmd.PersistentFlags().Lookup("grpc-address"))
	_ = viper.BindPFlag("certificate_file", healthCheckCmd.PersistentFlags().Lookup("certificate-file"))
	_ = viper.BindPFlag("common_name", healthCheckCmd.PersistentFlags().Lookup("common-name"))
}
