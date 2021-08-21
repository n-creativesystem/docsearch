package cmd

import (
	"fmt"

	"github.com/n-creativesystem/docsearch/version"
	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Long:  "Print the version number",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("version: %s\n", version.Version)
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}
