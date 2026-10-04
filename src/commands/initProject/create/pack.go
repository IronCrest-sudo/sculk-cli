package create

import (
	"encoding/json"
	"os"
	"path/filepath"

	"sculk-cli/src/version"
)

func CreatePackMcmeta(projectVersion string, projectType string) error {

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// CREATE pack.mcmeta
	pathName := filepath.Join(workingDir + "/pack.mcmeta")
	file, err := os.Create(pathName)
	if err != nil {
		panic(err)
	}

	// WRITE to pack.mcmeta
	writePackMcmeta(file, projectVersion, projectType)

	err = file.Close()
	return err
	// fmt.Println(pathName, projectVersion)
}

// Pack Format for Datapack / Resourcepack
// source: https://minecraft.wiki/w/Pack_format
//
// Only versions whose format number is known for certain are listed. Anything
// newer falls back to the closest known entry and the caller is told that the
// number is a guess, so a wrong pack_format never lands silently in a pack.
var datapackPackFormatMap = map[string]int{
	"26.4-snapshot-1": 122,
	"26.2":            107,
}

var resourcepackPackFormatMap = map[string]int{
	"26.2": 88,
}

// PackFormatFor looks up the pack format for a game version and pack kind
// ("datapack" or "resourcepack").
//
// exact is false when the version is not in the table and the closest known
// older entry was used instead; callers should warn the user in that case.
func PackFormatFor(gameVersion string, kind string) (packFormat int, exact bool) {
	table := datapackPackFormatMap
	if kind == "rp" || kind == "resourcepack" {
		table = resourcepackPackFormatMap
	}

	want := version.Parse(gameVersion)

	if v, ok := table[gameVersion]; ok {
		return v, true
	}

	// Closest known version that is not newer than the requested one.
	best := ""
	var bestVersion version.Version
	for candidate := range table {
		c := version.Parse(candidate)
		if c.Compare(want) > 0 {
			continue
		}
		if best == "" || c.Compare(bestVersion) > 0 {
			best, bestVersion = candidate, c
		}
	}
	if best == "" {
		// Nothing older known: use the oldest entry in the table.
		for candidate := range table {
			c := version.Parse(candidate)
			if best == "" || c.Compare(bestVersion) < 0 {
				best, bestVersion = candidate, c
			}
		}
	}
	if best == "" {
		return 0, false
	}
	return table[best], false
}

func writePackMcmeta(file *os.File, projectVersion string, projectType string) {

	var packFormatNumber int
	var packDescription string

	// Different pack_format value based on resourcepack/datapack
	if projectType == "dp" {
		packFormatNumber, _ = PackFormatFor(projectVersion, "dp")
		packDescription = "a Sculk Project [datapack]"
	} else {
		packFormatNumber, _ = PackFormatFor(projectVersion, "rp")
		packDescription = "a Sculk Project [resourcepack]"
	}

	// Construct pack.mcmeta
	packFileContent := PackMcmetaFileContent{
		PackData: Pack{
			Pack_format: packFormatNumber,
			Description: packDescription,
		},
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(packFileContent)
	if err != nil {
		panic(err)
	}
}
