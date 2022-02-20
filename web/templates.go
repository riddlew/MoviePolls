package web

import (
	"fmt"
	"html/template"
	"net/http"
)

const TEMPLATE_DIR = "web/templates/"
const TEMPLATE_BASE = TEMPLATE_DIR + "base.html"

// templateDefs is static throughout the life of the server process
var templateDefs map[string][]string = map[string][]string{
	"movieinfo":     []string{"movie-info.html"},
	"cyclevotes":    []string{"cycle.html"},
	"movieError":    []string{"movie-error.html"},
	"simplelogin":   []string{"plain-login.html"},
	"addmovie":      []string{"add-movie.html"},
	"account":       []string{"account.html"},
	"newaccount":    []string{"newaccount.html"},
	"error":         []string{"error.html"},
	"history":       []string{"history.html"},
	"auth":          []string{"auth.html"},
	"passwordReset": []string{"password.html"},

	"adminHome":      []string{"admin/base.html", "admin/home.html"},
	"adminConfig":    []string{"admin/base.html", "admin/config.html"},
	"adminUsers":     []string{"admin/base.html", "admin/users.html"},
	"adminUserEdit":  []string{"admin/base.html", "admin/user-edit.html"},
	"adminCycles":    []string{"admin/base.html", "admin/cycles.html"},
	"adminEndCycle":  []string{"admin/base.html", "admin/endcycle.html"},
	"adminMovies":    []string{"admin/base.html", "admin/movies.html"},
	"adminMovieEdit": []string{"admin/base.html", "admin/movie-edit.html"},
	"adminNotice":    []string{"admin/base.html", "admin/notice.html"},
	"adminConfirm":   []string{"admin/base.html", "admin/confirmation.html"},
}

func (s *webServer) registerTemplates() error {
	s.templates = make(map[string]*template.Template)

	for key, files := range templateDefs {
		fpth := []string{TEMPLATE_BASE}
		for _, f := range files {
			fpth = append(fpth, TEMPLATE_DIR+f)
		}

		t, err := template.ParseFiles(fpth...)
		if err != nil {
			return fmt.Errorf("Error parsing template %s: %v", fpth, err)
		}

		s.templates[key] = t
	}
	return nil
}

func (s *webServer) executeTemplate(w http.ResponseWriter, key string, data interface{}) error {
	// for deugging only
	if s.debug {
		err := s.registerTemplates()
		if err != nil {
			return err
		}
	}

	t, ok := s.templates[key]
	if !ok {
		return fmt.Errorf("Template with key %q does not exist", key)
	}

	if err := t.Execute(w, data); err != nil {
		return fmt.Errorf("[%s] %v", key, err)
	}

	return nil
}

func (s *webServer) newPageBase(title string, w http.ResponseWriter, r *http.Request) dataPageBase {
	cycle, err := s.backend.GetCurrentCycle()
	if err != nil {
		s.l.Error("[newPageBase] Unable to get current cycle: %v\n", err)
	}

	notice, err := s.backend.GetConfigBanner()
	if err != nil {
		s.l.Error("Unable to get notice message from database: %v", err)
	}

	return dataPageBase{
		PageTitle: title,
		Notice:    notice,

		User:         s.getSessionUser(w, r),
		CurrentCycle: cycle,
	}
}
