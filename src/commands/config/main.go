package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"charm.land/log/v2"
)

type Config struct {
	doMerge bool
	author Author
	initTemplate string
}

type Author struct {
	name string
}

func Main(args []string) {}

func InitConfig() {
	// config doesn't exist, create one.
	appDir, err := os.UserConfigDir()
	if err != nil {panic(err)}
	
	sculkConfigFilePath := filepath.Join(appDir, ".sculk", "config.json")
	sculkConfigFilePathDir := filepath.Dir(sculkConfigFilePath)
	err = os.MkdirAll(sculkConfigFilePathDir, 0644)
	if err != nil {panic(err)}
	_, err = os.Create(sculkConfigFilePath)
	if err != nil {panic(err)}
	
	sculkConfigBase := Config{
		doMerge: true,				// Merge to Existing Datapack
		author: Author{
			name: "Sculk Author",	// Author Name
		},
		initTemplate: "none",		// Init Template, incase other people have different file-structure practices.
	}
	sculkConfigBaseJson, err := json.Marshal(sculkConfigBase)
	// write to config.
	err = os.WriteFile(sculkConfigFilePath, sculkConfigBaseJson, 0755)
	if err != nil {panic(err)}
}
	
func GetSculkConfig() (configJsonData *Config, filePath string, error error) {
	appDir, err := os.UserConfigDir()
	if err != nil {panic(err)}

	sculkConfigFile := filepath.Join(appDir, ".sculk", "config.json")

	configDataFile, err := os.ReadFile(sculkConfigFile)
	if err != nil {
		return nil, "", errors.New("Config File Doesn't Exist.")
	}

	var configData Config
	err = json.Unmarshal(configDataFile, &configData)

	return &configData, sculkConfigFile, err
}

func ConfigExists() bool {
	_, _, err := GetSculkConfig()
	if err != nil {
		log.Print("Couldn't find config.json, created a new base config. Use 'sculk config' to change values.")
		return false
	} else {
		return true
	}
}



func setParamValue(param string, value string) {}