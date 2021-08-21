package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func initialize(before func(), after func()) func() {
	if before == nil {
		before = func() {}
	}
	if after == nil {
		after = func() {}
	}
	return func() {
		before()
		if configFile != "" {
			viper.SetConfigFile(configFile)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			viper.AddConfigPath("/etc")
			viper.AddConfigPath(home)
			viper.SetConfigName("docsearch")
		}

		viper.SetEnvPrefix("docsearch")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			switch err.(type) {
			case viper.ConfigFileNotFoundError:
				// config file does not found in search path
			default:
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		after()
	}
}
