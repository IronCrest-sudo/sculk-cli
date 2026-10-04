package makelib

import (
	"sculk-cli/src/commands/config"
	"sculk-cli/src/commands/initProject/create"

	"charm.land/log/v2"
)

// Main drops a libraries.json into an existing pack so that it can be
// consumed - and published - by sculk.
//
//	sculk makelib                 use the configured author and game version
//	sculk makelib <gameVersion>   pin a specific game version
func Main(args []string) error {
	authorName := config.AuthorName()

	gameVersion := "26.3"
	if len(args) > 0 && args[0] != "" {
		gameVersion = args[0]
	}

	log.Printf("🚧 Creating libraries.json for author '%s', Minecraft %s", authorName, gameVersion)
	create.CreateLibrariesJson(authorName, gameVersion)
	return nil
}
