package copilot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const maxTailBytes int64 = 64 * 1024

type ChatSnapshot struct {
	LogFile         string
	LatestUser      string
	LatestAssistant string
	RawExcerpt      string
}

type candidateFile struct {
	path    string
	modTime time.Time
}

// ReadLatestChat finds the newest Copilot chat log and extracts the latest messages.
func ReadLatestChat() (ChatSnapshot, error) {
	logFile, err := newestLogFile()
	if err != nil {
		return ChatSnapshot{}, err
	}

	excerpt, err := tailFile(logFile, maxTailBytes)
	if err != nil {
		return ChatSnapshot{}, fmt.Errorf("tail log: %w", err)
	}

	latestUser, latestAssistant := extractMessages(excerpt)

	return ChatSnapshot{
		LogFile:         logFile,
		LatestUser:      latestUser,
		LatestAssistant: latestAssistant,
		RawExcerpt:      excerpt,
	}, nil
}

func newestLogFile() (string, error) {
	roots := candidateRoots()
	files := make([]candidateFile, 0, 16)

	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.Contains(path, "GitHub.copilot-chat") {
				return nil
			}
			if !strings.Contains(path, "debug-logs") {
				return nil
			}
			info, statErr := d.Info()
			if statErr != nil {
				return nil
			}
			files = append(files, candidateFile{path: path, modTime: info.ModTime()})
			return nil
		})
	}

	if len(files) == 0 {
		return "", errors.New("copilot chat log not found; set COPILOT_LOG_ROOTS")
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	return files[0].path, nil
}

func candidateRoots() []string {
	if custom := strings.TrimSpace(os.Getenv("COPILOT_LOG_ROOTS")); custom != "" {
		parts := strings.Split(custom, ":")
		roots := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			roots = append(roots, expandHome(p))
		}
		return roots
	}

	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".config", "Code", "User", "workspaceStorage"),
		filepath.Join(home, ".config", "Code - Insiders", "User", "workspaceStorage"),
		filepath.Join(home, ".vscode-remote", "data", "User", "workspaceStorage"),
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func tailFile(path string, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	size := info.Size()
	start := int64(0)
	if size > maxBytes {
		start = size - maxBytes
	}

	if _, err := f.Seek(start, 0); err != nil {
		return "", err
	}

	var b strings.Builder
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		b.WriteString(scanner.Text())
		b.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func extractMessages(raw string) (string, string) {
	lines := strings.Split(raw, "\n")
	var latestUser, latestAssistant string

	re := regexp.MustCompile(`"(role|source|sender)"\s*:\s*"(user|assistant)"`)
	contentRE := regexp.MustCompile(`"(content|message|text)"\s*:\s*"([^"]+)"`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj map[string]any
		if json.Unmarshal([]byte(line), &obj) == nil {
			role := firstString(obj, "role", "source", "sender")
			msg := firstString(obj, "content", "message", "text")
			if role == "user" && msg != "" {
				latestUser = msg
			}
			if role == "assistant" && msg != "" {
				latestAssistant = msg
			}
			continue
		}

		if m := re.FindStringSubmatch(line); len(m) == 3 {
			role := m[2]
			if cm := contentRE.FindStringSubmatch(line); len(cm) == 3 {
				msg := cm[2]
				if role == "user" {
					latestUser = msg
				} else if role == "assistant" {
					latestAssistant = msg
				}
			}
		}
	}

	return cleanMessage(latestUser), cleanMessage(latestAssistant)
}

func firstString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := obj[key]; ok {
			if s, isString := v.(string); isString {
				return s
			}
		}
	}
	return ""
}

func cleanMessage(s string) string {
	s = strings.ReplaceAll(s, `\\n`, "\n")
	s = strings.ReplaceAll(s, `\\"`, `"`)
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		return s[:2000]
	}
	return s
}
