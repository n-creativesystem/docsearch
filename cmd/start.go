package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/n-creativesystem/docsearch/analyzer"
	"github.com/n-creativesystem/docsearch/client"
	"github.com/n-creativesystem/docsearch/config"
	"github.com/n-creativesystem/docsearch/helper"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "全文検索サーバーを起動します",
		Long:  "全文検索サーバーを起動します",
		RunE: func(cmd *cobra.Command, args []string) error {
			appConfig := config.New(config.WithNodeId(id), config.WithDirectory(dataDirectory))
			for _, dict := range userDictionaries {
				key := helper.GetFileNameWithoutExt(dict)
				if a, err := analyzer.UserDictionaryIsFile(dict); err != nil {
					return err
				} else {
					analyzer.Register(key, a)
				}
			}
			logrusLevel, err := logrus.ParseLevel(logLevel)
			if err != nil {
				logrusLevel = logrus.DebugLevel
			}
			logger := newLogger(logFormat, logrusLevel, logFile)
			bootstrap := peerGrpcAddress == "" || peerGrpcAddress == grpcAddress
			raftServer, err := server.NewRaftServer(appConfig, raftAddress, bootstrap, logger)
			if err != nil {
				return err
			}
			grpcServer, err := server.NewGRPCServerWithTLS(grpcAddress, raftServer, certificateFile, keyFile, commonName, logger)
			if err != nil {
				return err
			}
			grpcGateway, err := server.NewGRPCGateway(httpAddress, grpcAddress, certificateFile, keyFile, commonName, logger)
			if err != nil {
				return err
			}
			quitCh := make(chan os.Signal, 1)
			signal.Notify(quitCh, signals...)
			if err := raftServer.Start(); err != nil {
				return err
			}

			if err := grpcServer.Start(); err != nil {
				return err
			}

			if err := grpcGateway.Start(); err != nil {
				return err
			}

			if bootstrap {
				timeout := 60 * time.Second
				if err := raftServer.WaitForDetectLeader(timeout); err != nil {
					return err
				}
			}

			var joinGrpcAddress string
			if bootstrap {
				joinGrpcAddress = grpcAddress
			} else {
				joinGrpcAddress = peerGrpcAddress
			}

			c, err := client.NewGRPCClientWithContextTLS(joinGrpcAddress, context.Background(), certificateFile, commonName)
			if err != nil {
				return err
			}
			defer func() {
				_ = c.Close()
			}()

			// join this node to the existing cluster
			joinRequest := &protobuf.JoinRequest{
				Id: id,
				Node: &protobuf.Node{
					RaftAddress: raftAddress,
					Metadata: &protobuf.Metadata{
						GrpcAddress: grpcAddress,
						HttpAddress: httpAddress,
					},
				},
			}
			if err = c.Join(joinRequest); err != nil {
				if !bootstrap {
					return err
				}
			}
			logger.Info("docserach start")
			// wait for receiving signal
			<-quitCh

			_ = grpcGateway.Stop()
			_ = grpcServer.Stop()
			_ = raftServer.Stop()

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(startCmd)
	cobra.OnInitialize(initialize(nil, nil))
	allLogLevel := make([]string, len(logrus.AllLevels))
	for i, level := range logrus.AllLevels {
		allLogLevel[i] = level.String()
	}
	startCmd.PersistentFlags().StringVar(&configFile, "config-file", "", "config file. if omitted, docsearch.yaml in /etc and home directory will be searched")
	startCmd.PersistentFlags().StringVar(&id, "id", "node1", "node ID")
	startCmd.PersistentFlags().StringVar(&raftAddress, "raft-address", "127.0.0.1:7000", "Raft server listen address")
	startCmd.PersistentFlags().StringVar(&grpcAddress, "grpc-address", "127.0.0.1:9000", "gRPC server listen address")
	startCmd.PersistentFlags().StringVar(&httpAddress, "http-address", "127.0.0.1:8000", "HTTP server listen address")
	startCmd.PersistentFlags().StringVar(&dataDirectory, "data-directory", "data/docsearch", "data directory which store the index and Raft logs")
	startCmd.PersistentFlags().StringVar(&peerGrpcAddress, "peer-grpc-address", "", "listen address of the existing gRPC server in the joining cluster")
	startCmd.PersistentFlags().StringVar(&certificateFile, "certificate-file", "", "path to the client server TLS certificate file")
	startCmd.PersistentFlags().StringVar(&keyFile, "key-file", "", "path to the client server TLS key file")
	startCmd.PersistentFlags().StringVar(&commonName, "common-name", "", "certificate common name")
	startCmd.PersistentFlags().StringVar(&logFile, "log-file", "/dev/stdout", "log file name")
	startCmd.PersistentFlags().StringVar(&logLevel, "log-level", "DEBUG", fmt.Sprintf("log level[%s]", strings.Join(allLogLevel, ",")))
	startCmd.PersistentFlags().StringVar(&logFormat, "log-format", "json", "output log format")
	startCmd.PersistentFlags().StringArrayVar(&userDictionaries, "user-dictionary", nil, "")

	_ = viper.BindPFlag("id", startCmd.PersistentFlags().Lookup("id"))
	_ = viper.BindPFlag("raft_address", startCmd.PersistentFlags().Lookup("raft-address"))
	_ = viper.BindPFlag("grpc_address", startCmd.PersistentFlags().Lookup("grpc-address"))
	_ = viper.BindPFlag("http_address", startCmd.PersistentFlags().Lookup("http-address"))
	_ = viper.BindPFlag("data_directory", startCmd.PersistentFlags().Lookup("data-directory"))
	_ = viper.BindPFlag("peer_grpc_address", startCmd.PersistentFlags().Lookup("peer-grpc-address"))
	_ = viper.BindPFlag("certificate_file", startCmd.PersistentFlags().Lookup("certificate-file"))
	_ = viper.BindPFlag("key_file", startCmd.PersistentFlags().Lookup("key-file"))
	_ = viper.BindPFlag("common_name", startCmd.PersistentFlags().Lookup("common-name"))
	_ = viper.BindPFlag("log_file", startCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log_level", startCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("log_format", startCmd.PersistentFlags().Lookup("log-format"))
	_ = viper.BindPFlag("user_dictionary", startCmd.PersistentFlags().Lookup("user-dictionary"))
}
