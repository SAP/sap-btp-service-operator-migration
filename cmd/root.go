/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"os"
	"path/filepath"

	config "github.com/SAP/sap-btp-service-operator-migration/configuartion"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile, kubeconfig, managedNamespace string
	migrationConfig                       *config.Configuration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migration tool for migrating SVCAT to SAP BTP Service Operator.",
	Long:  `Migration tool from SVCAT to SAP BTP Service Operator.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(cmd.Help())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	// Here you will define your flags and configuration MigrationConfig.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&managedNamespace, "namespace", "n", "", "namespace to find operator secret (default sap-btp-operator)")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.migrate/config.json)")
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "absolute path to the kubeconfig file (default $HOME/.kube/config)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		cfgFile = filepath.Join(homeDir(), ".migrate", "config.json")

		// Search config in home directory with name ".migrate/config.json"
		viper.SetConfigFile(cfgFile)
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err == nil { // config file exist
		if managedNamespace != "" || kubeconfig != "" {
			// override config file
			createOrOverrideConfig()
		}
	} else {
		createOrOverrideConfig()
	}

	migrationConfig = config.NewConfiguration(context.Background(), viper.GetViper())
}

func createOrOverrideConfig() {
	err := ensureDirExists(cfgFile)
	cobra.CheckErr(err)
	kube := kubeconfig
	if kube == "" {
		kube = filepath.Join(homeDir(), ".kube", "config")
		if _, err := os.Stat(kube); os.IsNotExist(err) {
			kube = ""
		}
	}
	ns := managedNamespace
	if ns == "" {
		ns = "sap-btp-operator"
	}
	viper.Set("kubeconfig", kube)
	viper.Set("managedNamespace", ns)

	cobra.CheckErr(viper.WriteConfig())
}

func homeDir() string {
	home, err := homedir.Dir()
	cobra.CheckErr(err)
	return home
}

func ensureDirExists(path string) error {
	dirPath := filepath.Dir(path)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if mkderr := os.Mkdir(dirPath, 0700); mkderr != nil {
			return mkderr
		}
	}
	return nil
}
