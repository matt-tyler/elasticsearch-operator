// Copyright © 2017 Matt Tyler <me@matthewtyler.io>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	. "github.com/matt-tyler/elasticsearch-operator/e2e/pkg/run"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string
var config Config

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "e2e",
	Short: "Utility for running e2e tests for elasticsearch-operator",
	Run: func(cmd *cobra.Command, args []string) {
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Println(err.Error())
		}
		if err := Run(config, args); err != nil {
			fmt.Println(err.Error())
		}
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.e2e.yaml)")

	RootCmd.PersistentFlags().BoolP("build", "", false, "Build test executable")
	viper.BindPFlag("build", RootCmd.PersistentFlags().Lookup("build"))

	RootCmd.PersistentFlags().BoolP("up", "", false, "Spin up cluster on GKE")
	viper.BindPFlag("up", RootCmd.PersistentFlags().Lookup("up"))

	RootCmd.PersistentFlags().BoolP("down", "", false, "Tear down cluster after tests finish")
	viper.BindPFlag("down", RootCmd.PersistentFlags().Lookup("down"))

	RootCmd.PersistentFlags().BoolP("test", "", false, "Run tests")
	viper.BindPFlag("test", RootCmd.PersistentFlags().Lookup("test"))

	viper.BindEnv("PROJECT")

	viper.BindEnv("ZONE")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".e2e" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".e2e")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
