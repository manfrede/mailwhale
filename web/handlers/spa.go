package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

type SPAHandler struct {
	StaticPath      string
	IndexPath       string
	ReplaceBasePath string
	NoCache         bool
	indexContent    []byte
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Join internally call path.Clean to prevent directory traversal
	path := filepath.Join(h.StaticPath, r.URL.Path)

	// check whether a file exists at the given path
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.StaticPath, h.IndexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.FileServer(http.Dir(h.StaticPath)).ServeHTTP(w, r)
}

// dirty little hack to replace base attribute in html according to instance's public url
func (h *SPAHandler) loadIndex() {
	raw, err := os.ReadFile(filepath.Join(h.StaticPath, h.IndexPath))
	if err != nil {
		panic(err)
	}
	html := string(raw)
	if h.ReplaceBasePath != "" {
		pattern := regexp.MustCompile(`<base href="(.*)"`)
		html = pattern.ReplaceAllString(html, fmt.Sprintf(`<base href="%s"`, h.ReplaceBasePath))
	}
	h.indexContent = []byte(html)
}
