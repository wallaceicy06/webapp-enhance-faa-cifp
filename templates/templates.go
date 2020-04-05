package templates

import (
	"html/template"
	"path/filepath"

	_ "github.com/wallaceicy06/webapp-enhance-faa-cifp/alwaysroot"
)

var Base = template.Must(template.ParseFiles(filepath.Join("templates/base.html")))
