package server

import (
	"html/template"
	"net/http"
	"os"

	"github.com/hhhug000/harvester/crawler"
)

type Server struct {
	mux     *http.ServeMux
	tmpl    *template.Template
	crawler *crawler.Engine
}

type PageData struct {
	Domains []string
}

func NewServer(templateDir string, crawlerEngine *crawler.Engine) (*Server, error) {
	filesys := os.DirFS(templateDir)

	tmpls, err := template.ParseFS(filesys, "*.html", "*/*.html", "*/*/*.html")
	if err != nil {
		tmpls, err = template.ParseFS(filesys, "*.html")
		if err != nil {
			return nil, err
		}
	}

	s := &Server{
		mux:     http.NewServeMux(),
		tmpl:    tmpls,
		crawler: crawlerEngine,
	}

	s.Handle("/", s.HandleHome)

	return s, nil
}

func (s *Server) Render(w http.ResponseWriter, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := s.tmpl.ExecuteTemplate(w, templateName, data)
	if err != nil {
		http.Error(w, "Template Render Error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) Handle(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := PageData{
		Domains: []string{"golang.org", "pkg.go.dev", "go.dev"},
	}
	s.Render(w, "index.html", data)
}
