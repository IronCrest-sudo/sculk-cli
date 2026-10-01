// important data
package create

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/log/v2"
)

// Defines pack.mcmeta format
type PackMcmetaFileContent struct {
	PackData Pack `json:"pack"`
}

type Pack struct {
	Pack_format int    `json:"pack_format"`
	Description string `json:"description"`
}

type DatapackFile struct {
	DirName  string
	FileName string
	Content  []byte
}

type ResourcepackFile struct {
	DirName  string
	FileName string
	Content  []byte
}

type LibrariesDotJson struct {
	Name        string    `json:"name"`         // Name of the Project.
	Author      string    `json:"author"`       // Name of Author
	Version     string    `json:"version"`      // version of the sculk project
	GameVersion string    `json:"game_version"` // game version the sculk project is meant for - used to check versions while installing
	Libraries   []Library `json:"libraries"`    // All Installed Libraries
}

type Library struct {
	Identifier  string `json:"identifier"`   // Unique Identifier, handy for extremely popular packages.
	Source      string `json:"source"`       // A Git Repo Hosting Platform.
	Version     string `json:"version"`      // version of the imported library.
	GameVersion string `json:"game_version"` // version of the game the library supports.
}

func CreateLibrariesJson(author string, gameVersion string) {

	log.Printf("🚧  Creating libraries.json ...")
	// get target path
	targetPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// create libraries.json
	path := filepath.Join(targetPath + "/libraries.json")
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	// close file after program exits
	defer file.Close()
	// file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "   ")

	data := LibrariesDotJson{
		Author:      author,
		Version:     "1.0.0",
		GameVersion: gameVersion,
		Libraries:   []Library{},
	}

	// Write to JSON.
	encoder.Encode(data)
	log.Printf("✅  Created libraries.json")
}

func AddProjectNamespace(filePathPrefix string, namespace string, filePathSuffix string) string {
	return filePathPrefix + namespace + filePathSuffix
}

func CreateDatapackFiles(files []DatapackFile) {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// make them.
	for i := range files {
		err := os.MkdirAll(filepath.Join(workingDir, files[i].DirName), 0755)
		if err != nil {
			panic(err)
		}

		fullFilePath := filepath.Join(workingDir, files[i].DirName, files[i].FileName)

		file, err := os.Create(filepath.Join(fullFilePath))
		if err != nil {
			panic(err)
		}
		// write to file if buffer exists.
		if len(files[i].Content) > 0 {
			err := os.WriteFile(fullFilePath, files[i].Content, 0755)
			if err != nil {
				panic(err)
			}
		}
		file.Close()
	}
}

func CreateResourcepackFiles(files []ResourcepackFile) {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// make them.
	for i := range files {
		err := os.MkdirAll(filepath.Join(workingDir, files[i].DirName), 0755)
		if err != nil {
			panic(err)
		}

		// put into variable
		fullFilePath := filepath.Join(workingDir, files[i].DirName, files[i].FileName)

		file, err := os.Create(filepath.Join(workingDir, files[i].DirName, files[i].FileName))
		if err != nil {
			panic(err)
		}
		// write to file if buffer exists.
		if len(files[i].Content) > 0 {
			err := os.WriteFile(fullFilePath, files[i].Content, 0755)
			if err != nil {
				panic(err)
			}
		}
		file.Close()
	}
}

func SemanticVersionAdapting(versionString string, semanticVersionString string) bool {
	// versionString = "26.3"
	// semanticVersionString = ">=26.2"
	// versionString > semanticVersionString; return true.

	compareOperators := []string{
		">",		// greater than current version
		"<",		// older than current version
		">=",		// greater than or equal to
		"<=",		// older than or equal to
		// "~",		// approximately the same version (sub-version comparison)
		"=",		// exactly the same version (ignore)
	}

	compareOperator := ""
	for _, op := range compareOperators {
		if strings.HasPrefix(semanticVersionString, op) {
			compareOperator = op
			break
		}
	}

	// cut compare operator (">=26.3" => "26.3")
	semanticVersionStringSlice, found := strings.CutPrefix(semanticVersionString, compareOperator)
	if found == false {
		log.Print("No Comparison Operator found, treating it as '='.")
		compareOperator = "="
	}
	
	// support multiple-versioning (26.1.2 etc.)
	// "26.1" => ["26", "1"]
	// "26.1.2" => ["26", "1", "2"]
	versionStringList := strings.Split(versionString, ".")
	semanticVersionStringList := strings.Split(semanticVersionStringSlice, ".")

	var longestVersionString []string
	if len(versionStringList) > len(semanticVersionStringList) {
		longestVersionString = versionStringList
	} else {
		longestVersionString = semanticVersionStringList
	}
	
	for i, val := range longestVersionString {
		parsedSemanticVerVal, err := strconv.Atoi(val)
		if err != nil {return false}
		parsedVerVal, err := strconv.Atoi(versionStringList[i])
		if err != nil {return false}

		switch compareOperator {
			case "<":
				if parsedVerVal < parsedSemanticVerVal {
					return true
				}
			case ">":
				if parsedVerVal > parsedSemanticVerVal {
					return true
				}
			case "<=":
				if parsedVerVal <= parsedSemanticVerVal {
					return true
				}
			case ">=":
				if parsedVerVal >= parsedSemanticVerVal {
					return true
				}
			case "=":
				if parsedVerVal == parsedSemanticVerVal {
					return true
				}
		}
	}

	return false
}

func GetDatapackFilesList(projectName string, projectVersion string) []DatapackFile {
	fileStructure := map[string][]DatapackFile{
		"26.2": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"26.1": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.11": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.10": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.9": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.8": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.7": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.6": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.5": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.4": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.3": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.2": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21.1": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.21": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.20.6": {
			{DirName: "/data/minecraft/tags/functions/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/functions/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.20.5": {
			{DirName: "/data/minecraft/tags/functions/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/functions/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.20.4": {
			{DirName: "/data/minecraft/tags/functions/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/functions/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.20.3": {
			{DirName: "/data/minecraft/tags/functions/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/functions/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.20.1": {
			{DirName: "/data/minecraft/tags/functions/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/functions/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
		"1.20": {
			{DirName: "/data/minecraft/tags/functions/", FileName: "load.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/load\"") + "]}")},
			{DirName: "/data/minecraft/tags/functions/", FileName: "tick.json", Content: []byte(`{"values":[` + AddProjectNamespace("\"", projectName, ":global/tick\"") + "]}")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/functions/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
	}

	// for k := range fileStructure {
	// 	if SemanticVersionAdapting(projectVersion, k) {
	// 		return fileStructure[k]
	// 	}
	// }

	return fileStructure[projectVersion]
}

func GetResourcepackFilesList(projectName string, projectVersion string) []ResourcepackFile {
	fileStructure := map[string][]ResourcepackFile{
		"26.2": {
			{DirName: "/data/minecraft/tags/function/", FileName: "load.json", Content: []byte(`{"values":[]}`)},
			{DirName: "/data/minecraft/tags/function/", FileName: "tick.json", Content: []byte(`{"values":[]}`)},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "load.mcfunction", Content: []byte("")},
			{DirName: AddProjectNamespace("/data/", projectName, "/function/global/"), FileName: "tick.mcfunction", Content: []byte("")},
		},
	}
	return fileStructure[projectVersion]
}
