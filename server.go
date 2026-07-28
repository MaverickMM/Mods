package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type Mod struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type Manifest struct {
	AppID         string   `json:"app_id,omitempty"`
	WorkshopItems []string `json:"workshop_items,omitempty"`
	Mods          []Mod    `json:"mods"`
}

type collectionResponse struct {
	Response struct {
		CollectionDetails []struct {
			Result   int `json:"result"`
			Children []struct {
				PublishedFileID string `json:"publishedfileid"`
			} `json:"children"`
		} `json:"collectiondetails"`
	} `json:"response"`
}

type fileTask struct {
	path      string
	slashPath string
	publicURL string
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

	// 1. Ensure strict .gitignore exists to protect config/binaries
	ensureGitIgnore(exeDir)

	// 2. Ensure Git is available and check remote repository connectivity
	if err := ensureGitInstalled(); err != nil {
		fmt.Printf("❌ Git setup error: %v\n", err)
		return
	}

	if cfg.GitHubUser != defaultUser && cfg.GitHubRepo != defaultRepo {
		if err := ensureGitHubRepo(exeDir, cfg.GitHubUser, cfg.GitHubRepo); err != nil {
			fmt.Printf("⚠️ Remote repository setup error: %v\n", err)
		}
	} else {
		fmt.Println("ℹ️ Standard defaults detected in config.json. Update 'github_user' and 'github_repo' to automate GitHub sync.")
	}

	// 3. Construct public URL base path
	cleanDirName := filepath.ToSlash(cfg.ServerModsDir)
	cleanDirName = strings.TrimPrefix(cleanDirName, "./")
	cleanDirName = strings.TrimPrefix(cleanDirName, "/")
	publicBaseURL := fmt.Sprintf("https://%s.github.io/%s/%s", cfg.GitHubUser, cfg.GitHubRepo, cleanDirName)

	fmt.Println("=====================================")
	fmt.Println("        Mave Mod Synchronizer        ")
	fmt.Println("=====================================")
	fmt.Printf("Scanning folder : %s\n", targetModsDir)

	var tasks []fileTask
	seen := make(map[string]bool)

	// 4. Collect custom files from target mods directory
	if _, err := os.Stat(targetModsDir); err == nil {
		_ = filepath.Walk(targetModsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			fileName := info.Name()
			if strings.HasSuffix(fileName, ".tmp") || strings.HasPrefix(fileName, ".") {
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

			pathSegments := strings.Split(slashPath, "/")
			for i, seg := range pathSegments {
				pathSegments[i] = url.PathEscape(seg)
			}
			escapedURL := fmt.Sprintf("%s/%s", publicBaseURL, strings.Join(pathSegments, "/"))

			tasks = append(tasks, fileTask{
				path:      path,
				slashPath: slashPath,
				publicURL: escapedURL,
			})

			return nil
		})
	}

	// Hash local custom files in parallel across CPU threads
	scannedMods := hashFilesInParallel(tasks)
	sort.Slice(scannedMods, func(i, j int) bool {
		return scannedMods[i].Name < scannedMods[j].Name
	})

	fmt.Printf("Scanned %d custom file(s).\n", len(scannedMods))

	// 5. Extract Workshop Items and Resolve Workshop Collections
	workshopSet := make(map[string]bool)

	for _, raw := range cfg.WorkshopItems {
		if id := extractSteamID(raw); id != "" {
			workshopSet[id] = true
		}
	}

	if len(cfg.WorkshopCollections) > 0 {
		fmt.Printf("Resolving %d Workshop Collection(s)...\n", len(cfg.WorkshopCollections))
		for _, rawCol := range cfg.WorkshopCollections {
			colID := extractSteamID(rawCol)
			if colID == "" {
				continue
			}

			fmt.Printf(" 📚 Fetching items for Collection ID %s...\n", colID)
			items, err := fetchCollectionItemIDs(colID)
			if err != nil {
				fmt.Printf(" ⚠️ Failed to resolve collection %s: %v\n", colID, err)
				continue
			}

			for _, itemID := range items {
				workshopSet[itemID] = true
			}
			fmt.Printf("   └ Extracted %d item(s) from collection %s\n", len(items), colID)
		}
	}

	var resolvedWorkshopItems []string
	for itemID := range workshopSet {
		resolvedWorkshopItems = append(resolvedWorkshopItems, itemID)
	}
	sort.Strings(resolvedWorkshopItems)

	if len(resolvedWorkshopItems) > 0 {
		fmt.Printf("Total Workshop Item ID(s) compiled for AppID %s: %d\n", cfg.AppID, len(resolvedWorkshopItems))
	}

	// Guarantee slices render as empty JSON arrays [] instead of null
	if scannedMods == nil {
		scannedMods = []Mod{}
	}
	if resolvedWorkshopItems == nil {
		resolvedWorkshopItems = []string{}
	}

	// 6. Build Manifest JSON object
	manifest := Manifest{
		AppID:         cfg.AppID,
		WorkshopItems: resolvedWorkshopItems,
		Mods:          scannedMods,
	}

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

	// 7. Auto-commit and push updates to GitHub using standard Git
	if cfg.GitHubUser != defaultUser && cfg.GitHubRepo != defaultRepo {
		fmt.Println("Uploading updates to GitHub...")
		if err := pushToGitHub(exeDir, cfg.GitHubUser, cfg.GitHubRepo); err != nil {
			fmt.Printf("⚠️ Git sync skipped/failed: %v\n", err)
			return
		}
		fmt.Println("Done. Manifest and files are live!")
	}
}

// Multi-threaded file hashing for faster execution on large mods
func hashFilesInParallel(tasks []fileTask) []Mod {
	numWorkers := runtime.NumCPU() * 2
	if numWorkers > 16 {
		numWorkers = 16
	}

	taskChan := make(chan fileTask, len(tasks))
	resultChan := make(chan Mod, len(tasks))

	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskChan {
				hash, err := getHash(t.path)
				if err != nil {
					fmt.Printf(" ⚠️ Failed to hash file %s: %v\n", t.slashPath, err)
					continue
				}
				resultChan <- Mod{
					Name: t.slashPath,
					Hash: hash,
					URL:  t.publicURL,
				}
			}
		}()
	}

	wg.Wait()
	close(resultChan)

	var results []Mod
	for res := range resultChan {
		results = append(results, res)
	}
	return results
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

func extractSteamID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return ""
	}

	if matched, _ := regexp.MatchString(`^\d+$`, raw); matched {
		return raw
	}

	if u, err := url.Parse(raw); err == nil {
		if id := u.Query().Get("id"); id != "" {
			return id
		}
	}

	re := regexp.MustCompile(`id=(\d+)`)
	matches := re.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func fetchCollectionItemIDs(collectionID string) ([]string, error) {
	apiURL := "https://api.steampowered.com/ISteamRemoteStorage/GetCollectionDetails/v1/"

	formData := url.Values{}
	formData.Set("collectioncount", "1")
	formData.Set("publishedfileids[0]", collectionID)

	client := &http.Client{Timeout: 15 * time.Second}

	var resp *http.Response
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = client.PostForm(apiURL, formData)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("steam API network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API returned HTTP status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %w", err)
	}

	var parsed collectionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed parsing Steam collection JSON: %w", err)
	}

	if len(parsed.Response.CollectionDetails) == 0 {
		return nil, fmt.Errorf("collection ID %s not found", collectionID)
	}

	details := parsed.Response.CollectionDetails[0]
	if details.Result != 1 {
		return nil, fmt.Errorf("steam API error code %d (check if collection is public)", details.Result)
	}

	var itemIDs []string
	for _, child := range details.Children {
		if child.PublishedFileID != "" {
			itemIDs = append(itemIDs, child.PublishedFileID)
		}
	}

	return itemIDs, nil
}

func ensureGitIgnore(repoDir string) {
	ignorePath := filepath.Join(repoDir, ".gitignore")

	// Strict ignore rules: ignore everything except manifest.json, server_mods, and .gitignore
	content := "# 1. Ignore everything by default\n/*\n\n# 2. Whitelist required files\n!/manifest.json\n!.gitignore\n\n# 3. Whitelist server_mods and all contents\n!/server_mods/\n!/server_mods/**\n"
	_ = os.WriteFile(ignorePath, []byte(content), 0644)

	// Extra safety: force-remove .go source files from git index if staged
	_ = exec.Command("git", "rm", "--cached", "config.go", "git.go", "server.go", "go.mod", "go.sum").Run()
}
