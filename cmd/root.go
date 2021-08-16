package cmd

import (
	"os"
	"syscall"

	"github.com/ikawaha/blugeplugin/analysis/lang/ja"
	"github.com/n-creativesystem/docsearch/storage"
	"github.com/spf13/cobra"
)

var (
	configFile      string
	id              string
	raftAddress     string
	grpcAddress     string
	httpAddress     string
	dataDirectory   string
	peerGrpcAddress string
	certFile        string
	keyFile         string
	commonName      string
	file            string
	logLevel        string
	rootCmd         = &cobra.Command{
		Use:   "docsearch",
		Short: "分散全文検索サーバー",
		Long:  "分散全文検索サーバー",
	}
	signals = []os.Signal{
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGKILL,
		syscall.SIGTERM,
		syscall.SIGSTOP,
	}
)

func init() {
	storage.Register("ja", ja.Analyzer)
}

func Execute() error {
	return rootCmd.Execute()
}
