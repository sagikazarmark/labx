package labx

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

// Match labctl v0.1.112's archive metadata and .labctlignore semantics:
// https://github.com/iximiuz/labctl/blob/v0.1.112/cmd/content/archives.go
// https://github.com/iximiuz/labctl/blob/v0.1.112/cmd/content/push.go
func buildStartupArchive(output *os.Root, folder, archive string, compressed bool) error {
	folderFS, err := fs.Sub(output.FS(), folder)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	var writer io.Writer = &buf
	var gz *gzip.Writer
	if compressed {
		gz = gzip.NewWriter(&buf)
		writer = gz
	}
	tw := tar.NewWriter(writer)
	if err := archiveStartupDirectory(folderFS, tw, ".", nil); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			return err
		}
	}
	if existing, err := output.ReadFile(archive); err == nil && bytes.Equal(existing, buf.Bytes()) {
		return nil
	}
	if err := ensureOutputDir(output, path.Dir(archive)); err != nil {
		return err
	}
	return output.WriteFile(archive, buf.Bytes(), 0o644)
}

type startupIgnoreRule struct {
	directory string
	pattern   string
}

func archiveStartupDirectory(
	fsys fs.FS,
	tw *tar.Writer,
	directory string,
	inherited []startupIgnoreRule,
) error {
	rules := append([]startupIgnoreRule(nil), inherited...)
	data, err := fs.ReadFile(fsys, path.Join(directory, ".labctlignore"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			rules = append(rules, startupIgnoreRule{directory, line})
		}
	}
	entries, err := fs.ReadDir(fsys, directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := path.Join(directory, entry.Name())
		if ignoreStartupEntry(name, entry.IsDir(), rules) {
			continue
		}
		if entry.IsDir() {
			if err := archiveStartupDirectory(fsys, tw, name, rules); err != nil {
				return err
			}
			continue
		}
		if err := archiveStartupFile(fsys, tw, name); err != nil {
			return err
		}
	}
	return nil
}

func ignoreStartupEntry(name string, isDir bool, rules []startupIgnoreRule) bool {
	base := path.Base(name)
	if strings.HasPrefix(base, ".git") || strings.HasSuffix(base, "~") || base == ".labctlignore" {
		return true
	}
	for _, rule := range rules {
		if strings.HasSuffix(rule.pattern, "/") && !isDir {
			continue
		}
		pattern := strings.TrimSuffix(rule.pattern, "/")
		target := base
		if strings.Contains(pattern, "/") {
			target = name
			if rule.directory != "." {
				target = strings.TrimPrefix(name, rule.directory+"/")
			}
		}
		if matched, _ := path.Match(pattern, target); matched {
			return true
		}
	}
	return false
}

func archiveStartupFile(fsys fs.FS, tw *tar.Writer, name string) error {
	file, err := fsys.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:    name,
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		return err
	}
	_, err = io.Copy(tw, file)
	return err
}
