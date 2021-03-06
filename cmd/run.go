/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

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
	"github.com/SAP/sap-btp-service-operator-migration/migrate"
	"github.com/spf13/cobra"
)

var skipValidation *bool

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:     "run",
	Aliases: []string{"r"},
	Short:   "Run migration process",
	Long:    `Run migration process`,
	Run:     run,
}

func init() {
	rootCmd.AddCommand(runCmd)
	skipValidation = runCmd.Flags().BoolP("skip-validation", "s", false, "skip resources validation")
}

func run(_ *cobra.Command, _ []string) {
	ctx := migrationConfig.Context
	migrator := migrate.NewMigrator(ctx, migrationConfig.KubeConfig, migrationConfig.ManagedNamespace)
	execMode := migrate.Run
	if *skipValidation {
		execMode = migrate.RunWithoutValidation
	}
	migrator.Migrate(ctx, execMode)
}
