package engine

import (
	"path"
	"regexp"
	"strings"

	"github.com/swamp2k/copyarr/internal/config"
)

func pathAllowed(rel string, includes, excludes []string) bool {
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	for _, pattern := range includes {
		if filterMatch(pattern, rel) {
			return true
		}
	}
	for _, pattern := range excludes {
		if filterMatch(pattern, rel) {
			return false
		}
	}
	return true
}

func filterMatch(pattern, rel string) bool {
	pattern = strings.TrimSpace(strings.TrimPrefix(pattern, "/"))
	if pattern == "" {
		return false
	}
	if strings.HasSuffix(pattern, "/") {
		pattern += "**"
	}
	re, err := regexp.Compile(globRegex(pattern))
	if err == nil && re.MatchString(rel) {
		return true
	}
	// A pattern without a slash is a filename pattern and should match at any
	// depth, so "*.mkv" is useful without spelling "**/*.mkv".
	if !strings.Contains(pattern, "/") {
		re, err = regexp.Compile(globRegex(pattern))
		return err == nil && re.MatchString(path.Base(rel))
	}
	return false
}

func globRegex(pattern string) string {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			if strings.ContainsRune(`.+()|[]{}^$\\`, rune(ch)) {
				b.WriteByte('\\')
			}
			b.WriteByte(ch)
		}
	}
	b.WriteString("$")
	return b.String()
}

func rcloneFilterArgs(r config.Rule, relRoot string) []string {
	if len(r.Includes) == 0 && len(r.Excludes) == 0 {
		return nil
	}
	args := make([]string, 0, (len(r.Includes)+len(r.Excludes)+1)*2)
	for _, p := range r.Includes {
		if p = transferPattern(p, relRoot); p != "" {
			args = append(args, "--filter", "+ "+p)
		}
	}
	if len(r.Includes) > 0 {
		// Keep directory traversal open so a positive rule can rescue files
		// below a broad exclusion such as "**".
		args = append(args, "--filter", "+ */")
	}
	for _, p := range r.Excludes {
		if p = transferPattern(p, relRoot); p != "" {
			args = append(args, "--filter", "- "+p)
		}
	}
	return args
}

func transferPattern(pattern, relRoot string) string {
	pattern = strings.TrimSpace(strings.TrimPrefix(pattern, "/"))
	root := strings.Trim(strings.TrimSpace(relRoot), "/")
	if pattern == "" || root == "" {
		return pattern
	}
	if pattern == root {
		return "**"
	}
	if strings.HasPrefix(pattern, root+"/") {
		return strings.TrimPrefix(pattern, root+"/")
	}
	return pattern
}
