// THIS MODULE CREATES PACK.MCMETA
package create

import (
	"sculk-cli/src/commands/config"

	"charm.land/log/v2"
)

func CreateSculkProject(args []string, flags map[string]bool) error {

	projectName := args[0]
	projectVersion := args[1]
	log.Printf("🚧  Creating Sculk Project '%s' for Minecraft %s", projectName, projectVersion)

	projectType := "dp" // default is datapack project

	if flags["dp"] == true {
		projectType = "dp"
	} else if flags["rp"] == true {
		projectType = "rp"
	}

	// create libraries.json, stamped with the configured author name
	author := config.AuthorName()
	log.Printf("🚧  Author '%s' (change it with 'sculk config author <name>')", author)
	CreateLibrariesJson(author, projectVersion)

	// actually create files now
	switch projectType {
	case "dp":
		InitDatapack(projectName, projectVersion)
	case "rp":
		InitResourcepack(projectName, projectVersion)
	}

	// create pack.mcmeta file
	log.Printf("🚧  Creating pack.mcmeta ...")
	err := CreatePackMcmeta(projectVersion, projectType)
	log.Printf("✅  Created pack.mcmeta")
	return err
}
