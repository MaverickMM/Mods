package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ServerConfig struct {
	GitHubUser    string `json:"github_user"`
	GitHubRepo    string `json:"github_repo"`
	ServerModsDir string `json:"server_mods_dir"`
	OutputFile    string `json:"output_file"`
}

type Mod struct {
	Name string `json:"name"` // Stores relative path e.g. "config/options.json"
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type Manifest struct {
	Mods []Mod `json:"mods"`
}

const (
	defaultUser    = "your-username"
	defaultRepo    = "your-repo-name"
	defaultModsDir = "./server_mods"
	defaultOutput  = "./manifest.json"
)

func main() {
	exeDir := getExeDir()
	configFile := filepath.Join(exeDir, "config.json")

	cfg := loadServerConfig(configFile)

	targetModsDir := expandPath(cfg.ServerModsDir)
	if !filepath.IsAbs(targetModsDir) {
		targetModsDir = filepath.Join(exeDir, targetModsDir)
	}

	targetOutputFile := expandPath(cfg.OutputFile)
	if !filepath.IsAbs(targetOutputFile) {
		targetOutputFile = filepath.Join(exeDir, targetOutputFile)
	}

	publicBaseURL := fmt.Sprintf("https://%s.github.io/%s/mods", cfg.GitHubUser, cfg.GitHubRepo)

	fmt.Println("===================================")
	fmt.Println("   Mod Server Manifest Generator   ")
	fmt.Println("===================================")
	fmt.Printf("Scanning folder : %s\n", targetModsDir)

	manifest := Manifest{Mods: []Mod{}}
	seen := make(map[string]bool)

	// Walk recursively through targetModsDir to capture nested subdirectories
	err := filepath.Walk(targetModsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileName := info.Name()
		if info.IsDir() || strings.HasSuffix(fileName, ".tmp") || strings.HasPrefix(fileName, ".") {
			return nil
		}

		// Determine relative subpath from targetModsDir (e.g., "config/mod.json")
		relPath, err := filepath.Rel(targetModsDir, path)
		if err != nil {
			return nil
		}

		// Standardize separators to forward slashes for cross-platform manifest/URL consistency
		slashPath := filepath.ToSlash(relPath)

		if seen[slashPath] {
			fmt.Printf(" ⚠️ Duplicate ignored: %s\n", slashPath)
			return nil
		}
		seen[slashPath] = true

		hash := getHash(path)
		if hash != "" {
			// URL-escape path segments so subfolders/spaces stay valid
			pathSegments := strings.Split(slashPath, "/")
			for i, seg := range pathSegments {
				pathSegments[i] = url.PathEscape(seg)
			}
			escapedURL := fmt.Sprintf("%s/%s", publicBaseURL, strings.Join(pathSegments, "/"))

			manifest.Mods = append(manifest.Mods, Mod{
				Name: slashPath,
				Hash: hash,
				URL:  escapedURL,
			})
		}

		return nil
	})

	if err != nil {
		fmt.Printf("❌ Error reading mods directory: %v\n", err)
		return
	}

	sort.Slice(manifest.Mods, func(i, j int) bool {
		return manifest.Mods[i].Name < manifest.Mods[j].Name
	})

	fmt.Printf("Scanned %d file(s).\n", len(manifest.Mods))

	if err := os.MkdirAll(filepath.Dir(targetOutputFile), 0755); err != nil {
		fmt.Printf("❌ Error creating output directory: %v\n", err)
		return
	}

	outFile, err := os.Create(targetOutputFile)
	if err != nil {
		fmt.Printf("❌ Error creating manifest file: %v\n", err)
		return
	}

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		outFile.Close()
		fmt.Printf("❌ Error encoding JSON: %v\n", err)
		return
	}
	outFile.Close()

	fmt.Printf("Generated %s successfully.\n", filepath.Base(targetOutputFile))

	fmt.Println("Uploading updates to GitHub...")
	if err := pushToGitHub(exeDir); err != nil {
		fmt.Printf("⚠️ Git sync skipped/failed: %v\n", err)
		return
	}

	fmt.Println("Done. Changes are live!")
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

func pushToGitHub(repoDir string) error {
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository (missing .git folder in %s)", repoDir)
	}

	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoDir
	out, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v", err)
	}

	if len(out) == 0 {
		fmt.Println("No changes detected in working tree.")
		return nil
	}

	commitMsg := fmt.Sprintf("Auto update %s", time.Now().Format("2006-01-02 15:04:05"))

	commands := [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", commitMsg},
		{"git", "push", "origin", "main"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = repoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command failed (%s): %v", cmdArgs[0], err)
		}
	}

	return nil
}

func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func loadServerConfig(path string) ServerConfig {
	var cfg ServerConfig

	f, err := os.Open(path)
	if err != nil {
		cfg.GitHubUser = defaultUser
		cfg.GitHubRepo = defaultRepo
		cfg.ServerModsDir = defaultModsDir
		cfg.OutputFile = defaultOutput
		saveServerConfig(path, cfg)
		return cfg
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		fmt.Println("⚠️ Warning: Invalid config.json, using defaults.")
		cfg.GitHubUser = defaultUser
		cfg.GitHubRepo = defaultRepo
		cfg.ServerModsDir = defaultModsDir
		cfg.OutputFile = defaultOutput
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

func getHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}