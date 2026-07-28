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
	runCmd := func(args ...string) error {
		c := exec.Command("sudo", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		return c.Run()
	}

	switch {
	case commandExists("apt-get"):
		fmt.Printf("Installing %s via apt...\n", pkg)
		_ = runCmd("apt-get", "update")
		return runCmd("apt-get", "install", "-y", pkg)

	case commandExists("dnf"):
		fmt.Printf("Installing %s via dnf...\n", pkg)
		return runCmd("dnf", "install", "-y", pkg)

	case commandExists("pacman"):
		fmt.Printf("Installing %s via pacman...\n", pkg)
		return runCmd("pacman", "-S", "--noconfirm", pkg)

	case commandExists("zypper"):
		fmt.Printf("Installing %s via zypper...\n", pkg)
		return runCmd("zypper", "install", "-y", pkg)

	default:
		return fmt.Errorf("no supported package manager found (apt, dnf, pacman, zypper). Please install %s manually", pkg)
	}
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

func ensureGitIdentity(repoDir string) {
	nameCheck := exec.Command("git", "config", "user.name")
	nameCheck.Dir = repoDir
	if err := nameCheck.Run(); err != nil {
		setName := exec.Command("git", "config", "user.name", "Mave Synchronizer")
		setName.Dir = repoDir
		_ = setName.Run()
	}

	emailCheck := exec.Command("git", "config", "user.email")
	emailCheck.Dir = repoDir
	if err := emailCheck.Run(); err != nil {
		setEmail := exec.Command("git", "config", "user.email", "mave-bot@users.noreply.github.com")
		setEmail.Dir = repoDir
		_ = setEmail.Run()
	}
}

func ensureGitHubRepo(repoDir, user, repo string) error {
	gitDir := filepath.Join(repoDir, ".git")

	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Println("⚙️ Initializing local Git repository...")
		initCmd := exec.Command("git", "init")
		initCmd.Dir = repoDir
		if err := initCmd.Run(); err != nil {
			return fmt.Errorf("failed to run git init: %w", err)
		}

		branchCmd := exec.Command("git", "branch", "-M", "main")
		branchCmd.Dir = repoDir
		_ = branchCmd.Run()
	}

	ensureGitIdentity(repoDir)

	remoteURL := fmt.Sprintf("https://github.com/%s/%s.git", user, repo)

	remoteCheck := exec.Command("git", "remote", "get-url", "origin")
	remoteCheck.Dir = repoDir
	if err := remoteCheck.Run(); err != nil {
		fmt.Printf("🔗 Setting remote origin to %s...\n", remoteURL)
		addRemote := exec.Command("git", "remote", "add", "origin", remoteURL)
		addRemote.Dir = repoDir
		_ = addRemote.Run()
	}

	lsRemoteCmd := exec.Command("git", "ls-remote", "origin")
	lsRemoteCmd.Dir = repoDir
	if err := lsRemoteCmd.Run(); err != nil {
		fmt.Printf("⚠️ Remote repository %s/%s unreachable via Git.\n", user, repo)
		fmt.Println("   Make sure the remote repo exists on GitHub and your credentials/SSH keys are configured.")
		return nil
	}

	fmt.Println("✅ Connected to remote Git repository successfully.")
	return nil
}

func pushToGitHub(repoDir, user, repo string) error {
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := ensureGitHubRepo(repoDir, user, repo); err != nil {
			return err
		}
	}

	ensureGitIdentity(repoDir)

	// Clean up any interrupted rebase states
	rebaseFolder := filepath.Join(gitDir, "rebase-merge")
	rebaseApply := filepath.Join(gitDir, "rebase-apply")
	if _, err := os.Stat(rebaseFolder); err == nil {
		_ = exec.Command("git", "rebase", "--abort").Run()
		_ = os.RemoveAll(rebaseFolder)
	}
	if _, err := os.Stat(rebaseApply); err == nil {
		_ = exec.Command("git", "rebase", "--abort").Run()
		_ = os.RemoveAll(rebaseApply)
	}

	// Stage ALL changes including file DELETIONS
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = repoDir
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed staging files: %v", err)
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

	// Prefer local changes automatically if conflicts occur during sync
	commands := [][]string{
		{"git", "commit", "-m", commitMsg},
		{"git", "pull", "--rebase", "--autostash", "-X", "ours", "origin", currentBranch},
		{"git", "push", "-u", "origin", currentBranch},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = repoDir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command failed (%s): %v", cmdArgs[0], err)
		}
	}

	return nil
}
