package filesystem

import (
	"errors"
	"io/fs"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
)

type Service struct {
	fs   afero.Fs
	root string
}

type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"` // file | folder
	FileType string `json:"fileType,omitempty"`
	Children []Node `json:"children,omitempty"`
}

func NewService(root string) (*Service, error) {
	afs := afero.NewOsFs()
	return &Service{fs: afs, root: root}, nil
}

func (s *Service) abs(rel string) (string, error) {
	if rel == "" || rel == "/" {
		return s.root, nil
	}
	clean := filepath.Clean(rel)
	if strings.HasPrefix(clean, "..") {
		return "", errors.New("invalid path")
	}
	if strings.HasPrefix(clean, s.root) {
		return clean, nil
	}
	return filepath.Join(s.root, strings.TrimPrefix(clean, "/")), nil
}

func (s *Service) rel(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil || rel == "." {
		return "/"
	}
	return "/" + filepath.ToSlash(rel)
}

func (s *Service) detectFileType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".md":
		return "markdown"
	case ".db":
		return "database"
	case ".canvas":
		return "canvas"
	case ".html":
		return "html"
	case ".js", ".ts", ".tsx", ".jsx", ".go", ".py", ".rs", ".java":
		return "code"
	default:
		// try mime
		if mt := mime.TypeByExtension(ext); strings.Contains(mt, "text/") {
			return "code"
		}
		return "file"
	}
}

func (s *Service) ListTree(relPath string, depth int) (Node, error) {
	abs, err := s.abs(relPath)
	if err != nil {
		return Node{}, err
	}
	info, err := afero.ReadDir(s.fs, abs)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Node{}, err
	}
	// Create node for the directory itself
	dirName := filepath.Base(abs)
	if relPath == "" || relPath == "/" {
		dirName = "root"
	}
	node := Node{Name: dirName, Path: s.rel(abs), Type: "folder"}
	// Sort: folders first, then files, by name
	sort.Slice(info, func(i, j int) bool {
		if info[i].IsDir() && !info[j].IsDir() {
			return true
		}
		if !info[i].IsDir() && info[j].IsDir() {
			return false
		}
		return strings.ToLower(info[i].Name()) < strings.ToLower(info[j].Name())
	})
	for _, entry := range info {
		entryAbs := filepath.Join(abs, entry.Name())
		if entry.IsDir() {
			child := Node{Name: entry.Name(), Path: s.rel(entryAbs), Type: "folder"}
			if depth != 0 {
				sub, err := s.ListTree(s.rel(entryAbs), depth-1)
				if err == nil {
					child.Children = sub.Children
				}
			}
			node.Children = append(node.Children, child)
			continue
		}
		node.Children = append(node.Children, Node{
			Name:     entry.Name(),
			Path:     s.rel(entryAbs),
			Type:     "file",
			FileType: s.detectFileType(entry.Name()),
		})
	}
	return node, nil
}

func (s *Service) ReadFile(relPath string) (string, error) {
	abs, err := s.abs(relPath)
	if err != nil {
		return "", err
	}
	b, err := afero.ReadFile(s.fs, abs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Service) WriteFile(relPath string, content string) error {
	abs, err := s.abs(relPath)
	if err != nil {
		return err
	}
	if err := afero.WriteFile(s.fs, abs, []byte(content), 0o644); err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateFile(relPath, content string) error {
	abs, err := s.abs(relPath)
	if err != nil {
		return err
	}
	if err := s.ensureDir(filepath.Dir(abs)); err != nil {
		return err
	}
	return afero.WriteFile(s.fs, abs, []byte(content), 0o644)
}

func (s *Service) CreateFolder(relPath string) error {
	abs, err := s.abs(relPath)
	if err != nil {
		return err
	}
	return s.ensureDir(abs)
}

func (s *Service) ensureDir(abs string) error {
	return s.fs.MkdirAll(abs, 0o755)
}

func (s *Service) DeletePath(relPath string) error {
	abs, err := s.abs(relPath)
	if err != nil {
		return err
	}
	return s.fs.RemoveAll(abs)
}

func (s *Service) RenamePath(oldRel, newRel string) error {
	oldAbs, err := s.abs(oldRel)
	if err != nil {
		return err
	}
	newAbs, err := s.abs(newRel)
	if err != nil {
		return err
	}
	if err := s.ensureDir(filepath.Dir(newAbs)); err != nil {
		return err
	}
	return s.fs.Rename(oldAbs, newAbs)
}
