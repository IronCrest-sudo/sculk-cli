package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"charm.land/log/v2"
)

type Config struct {
	DoMerge      bool            `json:"doMerge"`
	Author       Author          `json:"author"`
	Sculk        SculkVersioning `json:"sculk"`
	InitTemplate string          `json:"initTemplate"`
}

type SculkVersioning struct {
	Version string `json:"version"`
}

type Author struct {
	Name string `json:"name"`
}

// Params is the ordered list of keys `sculk config` accepts, used both for
// validation and for the help/overview output.
var Params = []struct {
	Key         string
	Description string
	Example     string
}{
	{"doMerge", "merge libraries into the current project instead of installing them as a separate datapack", "sculk config doMerge false"},
	{"author", "default author name written into newly initialized libraries.json files", "sculk config author Barden"},
	{"initTemplate", "template applied by 'sculk init' ('none' disables it)", "sculk config initTemplate none"},
	{"sculk.version", "the sculk-cli version recorded in this config", "sculk config sculk.version 1.1.0"},
}

// paramKeys is the set of valid keys, lower-cased.
func paramKeys() map[string]bool {
	out := make(map[string]bool, len(Params))
	for _, p := range Params {
		out[strings.ToLower(p.Key)] = true
	}
	return out
}

// Main implements `sculk config [param] [value...]`.
//
//	sculk config                     show every value
//	sculk config <param>             show one value
//	sculk config <param> <value>     set a value
//	sculk config list                alias for showing every value
//	sculk config reset               rewrite the defaults
func Main(args []string) error {
	// A missing config is not an error: create one and carry on.
	ConfigExists()

	if len(args) == 0 {
		printAll()
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "list", "ls", "all", "--list", "-l":
		printAll()
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	case "reset", "--reset":
		InitConfig()
		log.Printf("🧹 Config reset to defaults (%s)", configFilePath())
		printAll()
		return nil
	}

	param := args[0]
	if !paramKeys()[strings.ToLower(param)] {
		return fmt.Errorf("unknown config parameter %q (valid: %s)", param, strings.Join(validKeys(), ", "))
	}

	// `sculk config <param>` with no value -> print it.
	if len(args) == 1 {
		cfg, _, err := GetSculkConfig()
		if err != nil {
			return err
		}
		fmt.Printf("%s = %s\n", param, get(cfg, param))
		return nil
	}

	value := strings.Join(args[1:], " ")
	if err := setParamValue(param, value); err != nil {
		return err
	}
	return nil
}

// validKeys returns the sorted list of accepted parameter names.
func validKeys() []string {
	keys := make([]string, 0, len(Params))
	for _, p := range Params {
		keys = append(keys, p.Key)
	}
	sort.Strings(keys)
	return keys
}

// setParamValue writes a single parameter back to the config file.
func setParamValue(param string, value string) error {
	cfg, path, err := GetSculkConfig()
	if err != nil {
		return err
	}

	switch strings.ToLower(param) {
	case "domerge":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("'doMerge' expects true or false, got %q", value)
		}
		cfg.DoMerge = b
	case "author":
		name := strings.TrimSpace(value)
		if name == "" {
			return errors.New("'author' cannot be empty")
		}
		cfg.Author.Name = name
	case "inittemplate":
		name := strings.TrimSpace(value)
		if name == "" {
			return errors.New("'initTemplate' cannot be empty")
		}
		cfg.InitTemplate = name
	case "sculk.version":
		v := strings.TrimSpace(value)
		if v == "" {
			return errors.New("'sculk.version' cannot be empty")
		}
		cfg.Sculk.Version = v
	default:
		return fmt.Errorf("unknown config parameter %q", param)
	}

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	log.Printf("⚙ %s = %s", param, get(cfg, param))
	log.Printf("📄 %s", path)
	return nil
}

// get reads a parameter back out of a Config, for display purposes.
func get(cfg *Config, param string) string {
	switch strings.ToLower(param) {
	case "domerge":
		return strconv.FormatBool(cfg.DoMerge)
	case "author":
		return cfg.Author.Name
	case "inittemplate":
		return cfg.InitTemplate
	case "sculk.version":
		return cfg.Sculk.Version
	}
	return ""
}

// parseBool accepts the usual spellings so that `doMerge 0` and
// `doMerge off` both work.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "t", "yes", "y", "on", "1", "enable", "enabled":
		return true, nil
	case "false", "f", "no", "n", "off", "0", "disable", "disabled":
		return false, nil
	}
	return false, fmt.Errorf("not a boolean: %q", s)
}

// SaveConfig writes cfg back to the config file, creating it if needed.
func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}
	path := configFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// configFilePath is the absolute path of the sculk config file.
func configFilePath() string {
	appDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(appDir, ".sculk", "config.json")
}

func InitConfig() {
	// config doesn't exist, create one.
	sculkConfigBase := Config{
		DoMerge: true, // Merge to Existing Datapack
		Author: Author{
			Name: "Sculk Author", // Author Name
		},
		Sculk:        SculkVersioning{Version: "1.0.0"},
		InitTemplate: "none", // Init Template, incase other people have different file-structure practices.
	}

	if err := SaveConfig(&sculkConfigBase); err != nil {
		panic(err)
	}
}

func GetSculkConfig() (configJsonData *Config, filePath string, error error) {
	sculkConfigFile := configFilePath()

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
		log.Print("Couldn't find config.json, creating a new base config. Use 'sculk config' to change values.")
		InitConfig()
		return false
	} else {
		return true
	}
}

// MustConfig returns the config, creating a default one first if needed.
// Every command that reads config should go through this rather than
// GetSculkConfig so that a fresh install never panics.
func MustConfig() *Config {
	ConfigExists()
	cfg, _, err := GetSculkConfig()
	if err != nil {
		log.Printf("⚠ Could not read config (%v), falling back to defaults.", err)
		return &Config{
			DoMerge:      true,
			Author:       Author{Name: "Sculk Author"},
			Sculk:        SculkVersioning{Version: "1.0.0"},
			InitTemplate: "none",
		}
	}
	return cfg
}

// ShouldMerge reports whether libraries are merged into the current project.
// It is the single place the doMerge setting is read, so that every command
// agrees on the behaviour.
func ShouldMerge() bool { return MustConfig().DoMerge }

// AuthorName returns the configured default author name.
func AuthorName() string { return MustConfig().Author.Name }

func printAll() {
	cfg := MustConfig()
	fmt.Printf("\nsculk config  (%s)\n\n", configFilePath())
	fmt.Printf("  %-14s %-8s  %s\n", "doMerge", strconv.FormatBool(cfg.DoMerge),
		"merge libraries into this project, or install them as separate datapacks")
	fmt.Printf("  %-14s %-8s  %s\n", "author", cfg.Author.Name, "default author for new libraries.json files")
	fmt.Printf("  %-14s %-8s  %s\n", "initTemplate", cfg.InitTemplate, "template used by 'sculk init'")
	fmt.Printf("  %-14s %-8s  %s\n", "sculk.version", cfg.Sculk.Version, "sculk-cli version recorded here")
	fmt.Printf("\n  change one with: sculk config <param> <value>\n\n")
}

func printHelp() {
	fmt.Printf("\nsculk config [param] [value]\n\n")
	for _, p := range Params {
		fmt.Printf("  %-14s %s\n", p.Key, p.Description)
		fmt.Printf("  %-14s e.g. %s\n\n", "", p.Example)
	}
	fmt.Printf("  %-14s %s\n", "list", "print every value")
	fmt.Printf("  %-14s %s\n", "reset", "rewrite the defaults")
	fmt.Println()
}
