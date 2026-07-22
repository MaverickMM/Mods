package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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
	cmd.Stdin = os.Stdin

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

	remoteURL := fmt.Sprintf("https://github.com/%s/%s.git", user, repo)

	remoteCheck := exec.Command("git", "remote", "get-url", "origin")
	remoteCheck.Dir = repoDir
	if err := remoteCheck.Run(); err != nil {
		fmt.Printf("Setting remote origin to %s...\n", remoteURL)
		addRemote := exec.Command("git", "remote", "add", "origin", remoteURL)
		addRemote.Dir = repoDir
		_ = addRemote.Run()
	}

	if runtime.GOOS == "linux" {
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