/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"sculk-cli/src/commands/config"

	"charm.land/log/v2"
	"github.com/spf13/cobra"
)

// onboardCmd represents the onboard command
var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Run this command on first-time download of sculk-cli (to initialize important stuff)",
	Long: `Run this command on first-time download of sculk-cli (to initialize important stuff)`,
	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()
		log.Printf("sculk-cli has been installed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}
