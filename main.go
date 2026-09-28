/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"sculk-cli/cmd"
	"sculk-cli/src/commands/config"
)

func main() {
	config.ConfigExists()
	cmd.Execute()
}
