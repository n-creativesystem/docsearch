package cmd

import (
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	configFile       string
	id               string
	raftAddress      string
	grpcAddress      string
	httpAddress      string
	dataDirectory    string
	peerGrpcAddress  string
	certificateFile  string
	keyFile          string
	commonName       string
	logFile          string
	logLevel         string
	logFormat        string
	userDictionaries []string
	rootCmd          = &cobra.Command{
		Use:   "docsearch",
		Short: "分散全文検索サーバー",
		Long:  "分散全文検索サーバー",
	}
	signals = []os.Signal{
		os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	}
)

func Execute() error {
	return rootCmd.Execute()
}
