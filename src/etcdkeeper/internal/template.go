package internal

import (
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type templateHandler struct {
	config    interface{}
	templates *template.Template
}

func NewTemplateServer(rootDir fs.FS, templateData interface{}) *templateHandler {

	root := template.New("")
	err := fs.WalkDir(rootDir, ".", func(currentPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			b, err := fs.ReadFile(rootDir, currentPath)
			if err != nil {
				return err
			}

			if len(currentPath) > 0 && !strings.HasPrefix(currentPath, "/") {
				currentPath = "/" + currentPath
			}
			_, err = root.New(currentPath).Parse(string(b))
			if err != nil {
				return err
			}
			log.Printf("template %q parsed\n", currentPath)
		}
		return nil
	})

	return &templateHandler{templateData, template.Must(root, err)}
}

// ServeHTTP check and serve template content if exist
func (h *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// this clean '..' '.' and trailing slash '/'
	cleanedURL := path.Clean(r.URL.Path)

	ext := path.Ext(cleanedURL)
	// if no extension on name this is a folder
	if ext == "" {
		cleanedURL = filepath.Join(cleanedURL, "/index.html")
	}

	err := h.templates.ExecuteTemplate(w, cleanedURL, h.config)
	if err == nil {
		log.Printf("GET : %s\n", cleanedURL)
	}
}
