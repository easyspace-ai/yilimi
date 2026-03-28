package tags

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Indexer builds and maintains a tag index from markdown files under a root directory.
// It supports:
//   - ReindexAll: full scan
//   - OnFsEvent: incremental updates on create/modify/delete/rename
//   - Query tags list and files by tag
type Indexer struct {
	root       string
	mu         sync.RWMutex
	tagToFiles map[string]map[string]int // tag -> file path -> count
	fileToTags map[string]map[string]int // file path -> tag -> count
	tagRegex   *regexp.Regexp
}

type TagCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type TagFileRef struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

func NewIndexer(root string) *Indexer {
	// Match inline tags. Support both '#' and '＃' and Unicode letters/numbers, underscores, hyphen, slash
	// Examples: #tag, #中文, #tag/sub, #long-tag_1
	tagRegex := regexp.MustCompile(`(?m)(?:^|[\s])[#＃]([\p{L}\p{N}_\-/]+)`) // capture group 1 is the tag
	return &Indexer{
		root:       root,
		tagToFiles: make(map[string]map[string]int),
		fileToTags: make(map[string]map[string]int),
		tagRegex:   tagRegex,
	}
}

// ReindexAll scans the entire root and rebuilds the index.
func (x *Indexer) ReindexAll() error {
	x.mu.Lock()
	x.tagToFiles = make(map[string]map[string]int)
	x.fileToTags = make(map[string]map[string]int)
	x.mu.Unlock()

	// Walk through root and index markdown files
	err := filepath.WalkDir(x.root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdown(p) {
			return nil
		}
		return x.indexFile(p)
	})
	return err
}

// OnFsEvent handles fs events to keep the index up-to-date.
// action one of: created | modified | deleted | renamed. For renamed, relPath is the destination.
func (x *Indexer) OnFsEvent(action string, absPath string) {
	// Only care markdown files
	if !isMarkdown(absPath) {
		return
	}
	switch action {
	case "created", "modified":
		_ = x.indexFile(absPath)
	case "deleted":
		x.removeFile(absPath)
	case "renamed":
		// Most watchers emit rename with old path. Treat as deletion; a subsequent create will index new path.
		x.removeFile(absPath)
	}
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

// indexFile parses a single markdown file and updates the index.
func (x *Indexer) indexFile(absPath string) error {
	// Read file
	b, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	content := string(b)
	tags := x.extractTags(content)

	// Update maps
	x.mu.Lock()
	defer x.mu.Unlock()

	// Clear previous
	if prev, ok := x.fileToTags[absPath]; ok {
		for t := range prev {
			if tf, ok := x.tagToFiles[t]; ok {
				delete(tf, absPath)
				if len(tf) == 0 {
					delete(x.tagToFiles, t)
				}
			}
		}
		delete(x.fileToTags, absPath)
	}

	if len(tags) == 0 {
		return nil
	}

	// Insert new
	fileMap := make(map[string]int)
	for tag, count := range tags {
		if _, ok := x.tagToFiles[tag]; !ok {
			x.tagToFiles[tag] = make(map[string]int)
		}
		x.tagToFiles[tag][absPath] = count
		fileMap[tag] = count
	}
	x.fileToTags[absPath] = fileMap
	return nil
}

func (x *Indexer) removeFile(absPath string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if prev, ok := x.fileToTags[absPath]; ok {
		for t := range prev {
			if tf, ok := x.tagToFiles[t]; ok {
				delete(tf, absPath)
				if len(tf) == 0 {
					delete(x.tagToFiles, t)
				}
			}
		}
		delete(x.fileToTags, absPath)
	}
}

// extractTags gets a map of tag -> count from content using frontmatter and inline tag syntax.
func (x *Indexer) extractTags(content string) map[string]int {
	result := map[string]int{}

	// Frontmatter between leading --- and next ---
	fmEnd := -1
	if strings.HasPrefix(content, "---\n") {
		// Find the end of frontmatter
		scanner := bufio.NewScanner(strings.NewReader(content))
		scanner.Split(bufio.ScanLines)
		lineNo := 0
		for scanner.Scan() {
			line := scanner.Text()
			if lineNo > 0 && strings.TrimSpace(line) == "---" {
				fmEnd = lineNo
				break
			}
			lineNo++
		}
		if fmEnd >= 0 {
			frontmatter := strings.Split(content, "\n")[:fmEnd]
			// Parse tags: [a, b] or multi-line - a
			for i := 0; i < len(frontmatter); i++ {
				line := strings.TrimSpace(frontmatter[i])
				if strings.HasPrefix(strings.ToLower(line), "tags:") {
					v := strings.TrimSpace(line[len("tags:"):])
					if strings.HasPrefix(v, "[") && strings.Contains(v, "]") {
						inside := strings.Trim(strings.TrimPrefix(v, "["), "]")
						items := strings.Split(inside, ",")
						for _, it := range items {
							t := normalizeTag(it)
							if t != "" {
								result[t]++
							}
						}
					} else if v == "" || v == "|" { // handle yaml multiline or folded
						// scan following indented lines starting with '-' until non-indented line
						j := i + 1
						for j < len(frontmatter) {
							l := strings.TrimSpace(frontmatter[j])
							if strings.HasPrefix(l, "-") {
								t := normalizeTag(strings.TrimSpace(strings.TrimPrefix(l, "-")))
								if t != "" {
									result[t]++
								}
								j++
								continue
							}
							break
						}
						i = j - 1
					}
				}
			}
		}
	}

	// Inline tags
	for _, m := range x.tagRegex.FindAllStringSubmatch(content, -1) {
		if len(m) >= 2 {
			t := normalizeTag(m[1])
			if t != "" {
				result[t]++
			}
		}
	}

	return result
}

func normalizeTag(s string) string {
	t := strings.TrimSpace(s)
	t = strings.Trim(t, "\"'")
	// Remove leading '#'
	t = strings.TrimLeft(t, "#＃")
	// Collapse multiple slashes
	t = strings.ReplaceAll(t, "//", "/")
	return t
}

// ListTags returns sorted tags by total count (sum across files).
func (x *Indexer) ListTags() []TagCount {
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := make([]TagCount, 0, len(x.tagToFiles))
	for tag, files := range x.tagToFiles {
		total := 0
		for _, c := range files {
			total += c
		}
		out = append(out, TagCount{Name: tag, Count: total})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	return out
}

// FilesForTag returns a list of files referencing the given tag, sorted by count desc.
func (x *Indexer) FilesForTag(tag string) []TagFileRef {
	x.mu.RLock()
	defer x.mu.RUnlock()
	files, ok := x.tagToFiles[tag]
	if !ok {
		return []TagFileRef{}
	}
	out := make([]TagFileRef, 0, len(files))
	for p, c := range files {
		out = append(out, TagFileRef{Path: toRelPath(x.root, p), Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Path < out[j].Path
		}
		return out[i].Count > out[j].Count
	})
	return out
}

// TagsForFile returns list of tags used in a file.
func (x *Indexer) TagsForFile(relOrAbsPath string) []string {
	// Normalize to absolute path
	abs := relOrAbsPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(x.root, strings.TrimPrefix(relOrAbsPath, "/"))
	}
	x.mu.RLock()
	defer x.mu.RUnlock()
	m := x.fileToTags[abs]
	if m == nil {
		return []string{}
	}
	out := make([]string, 0, len(m))
	for tag := range m {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func toRelPath(root string, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return "/" + filepath.ToSlash(rel)
}
