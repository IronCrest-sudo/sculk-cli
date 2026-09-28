package makelib

import (
	"sculk-cli/src/commands/initProject/create"
)

func Main(args []string) {

	authorName := "Sculk Author"
	gameVersion := "26.3"

	create.CreateLibrariesJson(authorName, gameVersion)
}