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
	"runtime"
	"sort"
	"strings"
	"time"
)

type Mod struct {
	Name string `json:"name"` // Stores relative path e.g. "config/options.json"
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type Manifest struct {
	Mods []Mod `json:"mods"`
}

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

	// 1. Ensure local tools (Git) and GitHub setup (gh CLI, repo) are ready
	if err := ensureGitInstalled(); err != nil {
		fmt.Printf("❌ Git setup error: %v\n", err)
		return
	}

	if cfg.GitHubUser != defaultUser && cfg.GitHubRepo != defaultRepo {
		if err := ensureGitHubRepo(exeDir, cfg.GitHubUser, cfg.GitHubRepo); err != nil {
			fmt.Printf("⚠️ Remote repository setup skipped: %v\n", err)
		}
	} else {
		fmt.Println("ℹ️ Standard defaults detected in config.json. Please update 'github_user' and 'github_repo' to automate remote sync.")
	}

	// 2. Build public base URL
	cleanDirName := strings.TrimPrefix(filepath.ToSlash(cfg.ServerModsDir), "./")
	publicBaseURL := fmt.Sprintf("https://%s.github.io/%s/%s", cfg.GitHubUser, cfg.GitHubRepo, cleanDirName)

	fmt.Println("=====================================")
	fmt.Println("         Mave Mod Synchronizer       ")
	fmt.Println("=====================================")
	fmt.Printf("Scanning folder : %s\n", targetModsDir)

	manifest := Manifest{Mods: []Mod{}}
	seen := make(map[string]bool)

	// 3. Scan directory and process files
	err := filepath.Walk(targetModsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileName := info.Name()
		if info.IsDir() || strings.HasSuffix(fileName, ".tmp") || strings.HasPrefix(fileName, ".") {
			return nil
		}

		relPath, err := filepath.Rel(targetModsDir, path)
		if err != nil {
			return nil
		}

		slashPath := filepath.ToSlash(relPath)

		if seen[slashPath] {
			fmt.Printf(" ⚠️ Duplicate ignored: %s\n", slashPath)
			return nil
		}
		seen[slashPath] = true

		hash, err := getHash(path)
		if err != nil {
			fmt.Printf(" ⚠️ Failed to hash file %s: %v\n", slashPath, err)
			return nil
		}

		// Properly escape individual path segments for URLs
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

	// 4. Save manifest output
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

	// 5. Upload to GitHub
	fmt.Println("Uploading updates to GitHub...")
	if err := pushToGitHub(exeDir, cfg.GitHubUser, cfg.GitHubRepo); err != nil {
		fmt.Printf("⚠️ Git sync skipped/failed: %v\n", err)
		return
	}

	fmt.Println("Done. Changes are live!")
}

// --- Cross-Platform Package Manager & Git Helpers ---

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func installLinuxPackage(pkg string) error {
	var cmd *exec.Cmd

	switch {
	case commandExists("apt-get"):
		fmt.Printf("Installing %s via apt...\n", pkg)
		cmd = exec.Command("sudo", "apt-get", "update")
		_ = cmd.Run()
		cmd = exec.Command("sudo", "apt-get", "install", "-y", pkg)

	case commandExists("dnf"):
		fmt.Printf("Installing %s via dnf...\n", pkg)
		cmd = exec.Command("sudo", "dnf", "install", "-y", pkg)

	case commandExists("pacman"):
		fmt.Printf("Installing %s via pacman...\n", pkg)
		cmd = exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg)

	case commandExists("zypper"):
		fmt.Printf("Installing %s via zypper...\n", pkg)
		cmd = exec.Command("sudo", "zypper", "install", "-y", pkg)

	default:
		return fmt.Errorf("no supported package manager found (apt, dnf, pacman, zypper). Please install %s manually", pkg)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Bind Stdin so users can type sudo password if prompted

	return cmd.Run()
}

func ensureGitInstalled() error {
	if commandExists("git") {
		return nil
	}

	fmt.Println("⚠️ Git not detected. Attempting automatic installation...")

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("winget", "install", "--id", "Git.Git", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("winget failed to install Git: %v", err)
		}
	case "linux":
		if err := installLinuxPackage("git"); err != nil {
			return fmt.Errorf("failed to install Git on Linux: %v", err)
		}
	default:
		return fmt.Errorf("git command not found. Please install Git manually on your OS")
	}

	fmt.Println("✅ Git installed successfully.")
	return nil
}

func ensureGHInstalled() error {
	// If running on Linux, skip checking or installing GitHub CLI entirely
	if runtime.GOOS == "linux" {
		return nil
	}

	if commandExists("gh") {
		return nil
	}

	fmt.Println("⚠️ GitHub CLI (gh) not found. Attempting automatic installation...")

	if runtime.GOOS == "windows" {
		cmd := exec.Command("winget", "install", "--id", "GitHub.cli", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("winget failed to install GitHub CLI: %v", err)
		}
		fmt.Println("✅ GitHub CLI installed successfully.")
		return nil
	}

	return fmt.Errorf("GitHub CLI (gh) is required on this OS")
}

func ensureGitHubRepo(repoDir, user, repo string) error {
	// Initialize local git repo if it doesn't exist yet
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Println("Initializing local Git repository...")
		initCmd := exec.Command("git", "init")
		initCmd.Dir = repoDir
		_ = initCmd.Run()

		branchCmd := exec.Command("git", "branch", "-M", "main")
		branchCmd.Dir = repoDir
		_ = branchCmd.Run()
	}

	// Determine the remote URL (Prefers SSH if git is configured for it, else HTTPS)
	remoteURL := fmt.Sprintf("https://github.com/%s/%s.git", user, repo)

	// Check if 'origin' remote is set locally; if not, add it
	remoteCheck := exec.Command("git", "remote", "get-url", "origin")
	remoteCheck.Dir = repoDir
	if err := remoteCheck.Run(); err != nil {
		fmt.Printf("Setting remote origin to %s...\n", remoteURL)
		addRemote := exec.Command("git", "remote", "add", "origin", remoteURL)
		addRemote.Dir = repoDir
		_ = addRemote.Run()
	}

	// -------------------------------------------------------------
	// LINUX / MANUAL GIT PATH (No `gh` CLI required)
	// -------------------------------------------------------------
	if runtime.GOOS == "linux" {
		// Test connection to the remote repo using native git
		lsRemoteCmd := exec.Command("git", "ls-remote", "origin")
		lsRemoteCmd.Dir = repoDir
		if err := lsRemoteCmd.Run(); err != nil {
			fmt.Printf("⚠️ Could not reach remote repository %s/%s via Git.\n", user, repo)
			fmt.Println("   Ensure the repository exists on GitHub and your SSH key / credentials are configured.")
			return nil
		}
		fmt.Println("✅ Connected to remote Git repository successfully.")
		return nil
	}

	// -------------------------------------------------------------
	// WINDOWS PATH (Uses `gh` CLI for browser login & repo creation)
	// -------------------------------------------------------------
	if err := ensureGHInstalled(); err != nil {
		return err
	}

	statusCmd := exec.Command("gh", "auth", "status")
	if err := statusCmd.Run(); err != nil {
		fmt.Println("🔑 You are not logged into GitHub CLI. Opening web login...")
		loginCmd := exec.Command("gh", "auth", "login", "--web")
		loginCmd.Stdin = os.Stdin
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr
		if err := loginCmd.Run(); err != nil {
			return fmt.Errorf("GitHub authentication failed: %v", err)
		}
	}

	repoFull := fmt.Sprintf("%s/%s", user, repo)
	viewCmd := exec.Command("gh", "repo", "view", repoFull)
	if err := viewCmd.Run(); err == nil {
		fmt.Printf("✅ Remote repository %s verified.\n", repoFull)
		return nil
	}

	fmt.Printf("🚀 Creating remote repository %s on GitHub...\n", repoFull)
	createCmd := exec.Command("gh", "repo", "create", repoFull, "--public", "--confirm")
	createCmd.Dir = repoDir
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create GitHub repository: %v", err)
	}

	return nil
}

func pushToGitHub(repoDir, user, repo string) error {
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository (missing .git folder in %s)", repoDir)
	}

	// Set remote origin if not already added
	remoteCheck := exec.Command("git", "remote", "get-url", "origin")
	remoteCheck.Dir = repoDir
	if err := remoteCheck.Run(); err != nil && user != defaultUser && repo != defaultRepo {
		remoteURL := fmt.Sprintf("https://github.com/%s/%s.git", user, repo)
		addRemote := exec.Command("git", "remote", "add", "origin", remoteURL)
		addRemote.Dir = repoDir
		_ = addRemote.Run()
	}

	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoDir
	out, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v", err)
	}

	if len(strings.TrimSpace(string(out))) == 0 {
		fmt.Println("No changes detected in working tree.")
		return nil
	}

	commitMsg := fmt.Sprintf("Auto update %s", time.Now().Format("2006-01-02 15:04:05"))

	// Detect current branch dynamically
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoDir
	branchOut, err := branchCmd.Output()
	currentBranch := "main"
	if err == nil {
		trimmed := strings.TrimSpace(string(branchOut))
		if trimmed != "" && trimmed != "HEAD" {
			currentBranch = trimmed
		}
	}

	commands := [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", commitMsg},
		{"git", "push", "-u", "origin", currentBranch},
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

func getHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}