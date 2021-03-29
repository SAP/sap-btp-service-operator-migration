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

// versionCmd represents the version command
var dryRunCmd = &cobra.Command{
	Use:     "dry-run",
	Aliases: []string{"d"},
	Short:   "Run migration in dry run mode",
	Long:    `Run only the validations, resources are not migrated`,
	Run:     dryRun,
}

func init() {
	rootCmd.AddCommand(dryRunCmd)
}

func dryRun(_ *cobra.Command, _ []string) {
	ctx := migrationConfig.Context
	migrator := migrate.NewMigrator(ctx, migrationConfig.KubeConfig, migrationConfig.ManagedNamespace)
	migrator.Migrate(ctx, migrate.DryRun)
}
