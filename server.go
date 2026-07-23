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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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

	// 1. Ensure local tools (Git) and GitHub setup are ready
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

	var scannedMods []Mod
	seen := make(map[string]bool)

	// 3. Scan directory and process local custom files
	if _, err := os.Stat(targetModsDir); err == nil {
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

			// Escape individual path segments for public URL construction
			pathSegments := strings.Split(slashPath, "/")
			for i, seg := range pathSegments {
				pathSegments[i] = url.PathEscape(seg)
			}
			escapedURL := fmt.Sprintf("%s/%s", publicBaseURL, strings.Join(pathSegments, "/"))

			scannedMods = append(scannedMods, Mod{
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

		sort.Slice(scannedMods, func(i, j int) bool {
			return scannedMods[i].Name < scannedMods[j].Name
		})
	}

	fmt.Printf("Scanned %d custom file(s).\n", len(scannedMods))

	// 4. Resolve Workshop Collections & Individual Items
	workshopSet := make(map[string]bool)

	// Process individual workshop items from config
	for _, raw := range cfg.WorkshopItems {
		id := extractSteamID(raw)
		if id != "" {
			workshopSet[id] = true
		}
	}

	// Process collections and fetch items via Steam API
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

	// Convert set to sorted slice
	var resolvedWorkshopItems []string
	for itemID := range workshopSet {
		resolvedWorkshopItems = append(resolvedWorkshopItems, itemID)
	}
	sort.Strings(resolvedWorkshopItems)

	if len(resolvedWorkshopItems) > 0 {
		fmt.Printf("Total Workshop Item ID(s) compiled for AppID %s: %d\n", cfg.AppID, len(resolvedWorkshopItems))
	}

	// 5. Construct complete Manifest object
	manifest := Manifest{
		AppID:         cfg.AppID,
		WorkshopItems: resolvedWorkshopItems,
		Mods:          scannedMods,
	}

	// Save manifest output
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

	// 6. Upload to GitHub
	fmt.Println("Uploading updates to GitHub...")
	if err := pushToGitHub(exeDir, cfg.GitHubUser, cfg.GitHubRepo); err != nil {
		fmt.Printf("⚠️ Git sync skipped/failed: %v\n", err)
		return
	}

	fmt.Println("Done. Changes are live!")
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

// Helper: Extract numeric ID from direct string or Steam URL
func extractSteamID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
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

// Helper: Query Steam Web API for items inside a collection
func fetchCollectionItemIDs(collectionID string) ([]string, error) {
	apiURL := "https://api.steampowered.com/ISteamRemoteStorage/GetCollectionDetails/v1/"

	formData := url.Values{}
	formData.Set("collectioncount", "1")
	formData.Set("publishedfileids[0]", collectionID)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm(apiURL, formData)
	if err != nil {
		return nil, fmt.Errorf("failed to query Steam API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API returned status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var parsed collectionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
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
