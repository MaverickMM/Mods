package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ServerConfig struct {
	GitHubUser    string   `json:"github_user"`
	GitHubRepo    string   `json:"github_repo"`
	ServerModsDir string   `json:"server_mods_dir"`
	OutputFile    string   `json:"output_file"`
	AppID         string   `json:"app_id"`
	WorkshopItems []string `json:"workshop_items"`
}

const (
	defaultUser    = "your-username"
	defaultRepo    = "your-repo-name"
	defaultModsDir = "./server_mods"
	defaultOutput  = "./manifest.json"
)

func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func loadServerConfig(path string) ServerConfig {
	var cfg ServerConfig

	f, err := os.Open(path)
	if err != nil {
		cfg = ServerConfig{
			GitHubUser:    defaultUser,
			GitHubRepo:    defaultRepo,
			ServerModsDir: defaultModsDir,
			OutputFile:    defaultOutput,
		}
		saveServerConfig(path, cfg)
		return cfg
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		fmt.Println("⚠️ Warning: Invalid config.json, using defaults.")
		cfg = ServerConfig{
			GitHubUser:    defaultUser,
			GitHubRepo:    defaultRepo,
			ServerModsDir: defaultModsDir,
			OutputFile:    defaultOutput,
		}
		return cfg
	}

	updated := false
	if cfg.GitHubUser == "" {
		cfg.GitHubUser = defaultUser
		updated = true
	}
	if cfg.GitHubRepo == "" {
		cfg.GitHubRepo = defaultRepo
		updated = true
	}
	if cfg.ServerModsDir == "" {
		cfg.ServerModsDir = defaultModsDir
		updated = true
	}
	if cfg.OutputFile == "" {
		cfg.OutputFile = defaultOutput
		updated = true
	}

	if updated {
		saveServerConfig(path, cfg)
	}

	return cfg
}

func saveServerConfig(path string, cfg ServerConfig) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(cfg)
}