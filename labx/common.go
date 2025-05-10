package labx

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/iximiuz/labctl/content"
)

func hasFiles(fsys fs.FS, kind content.ContentKind) (bool, error) {
	_, err := fs.Stat(fsys, fmt.Sprintf("dist/__static__/%s.tar.gz", kind.String()))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func createDownloadScript(kind content.ContentKind) string {
	targetDir := fmt.Sprintf("/opt/%s", kind)
	url := fmt.Sprintf("https://labs.iximiuz.com/__static__/%s.tar.gz?t=$(date +%%s)", kind)

	return fmt.Sprintf("mkdir -p %s\nwget --no-cache -O - \"%s\" | tar -xz -C %s", targetDir, url, targetDir)
}
