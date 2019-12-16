package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"
	//"time"
)

const TEMPLATE_DIR = "templates/"
const TEMPLATE_BASE = TEMPLATE_DIR + "base.html"

//var templates map[string]*template.Template

// templateDefs is static throughout the life of the server process
var templateDefs map[string][]string = map[string][]string{
	"movieinfo":   []string{"movie-info.html"},
	"cyclevotes":  []string{"cycle.html"},
	"movieError":  []string{"movie-error.html"},
	"simplelogin": []string{"plain-login.html"},
	"addmovie":    []string{"add-movie.html"},
	"account":     []string{"account.html"},
	"newaccount":  []string{"newaccount.html"},
	"error":       []string{"error.html"},

	"adminHome":     []string{"admin.html", "admin-home.html"},
	"adminConfig":   []string{"admin.html", "admin-config.html"},
	"adminUsers":    []string{"admin.html", "admin-users.html"},
	"adminUserEdit": []string{"admin.html", "admin-user-edit.html"},
}

func (s *Server) registerTemplates() error {
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

		fmt.Printf("Registering template %q\n", key)
		s.templates[key] = t
	}
	return nil
}

func (s *Server) executeTemplate(w http.ResponseWriter, key string, data interface{}) error {
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

	return t.Execute(w, data)
}

func (s *Server) newPageBase(title string, w http.ResponseWriter, r *http.Request) dataPageBase {
	return dataPageBase{
		PageTitle: title,
		User:      s.getSessionUser(w, r),
	}
}

type dataPageBase struct {
	PageTitle string

	User *User
}

type dataCycle struct {
	dataPageBase

	Cycle  *Cycle
	Movies []*Movie
}

type dataMovieInfo struct {
	dataPageBase

	Movie *Movie
}

type dataMovieError struct {
	dataPageBase
	ErrorMessage string
}

type dataLoginForm struct {
	dataPageBase
	ErrorMessage string
	Authed       bool
}

type dataAddMovie struct {
	dataPageBase
	ErrorMessage []string

	// Offending input
	ErrTitle       bool
	ErrDescription bool
	ErrLinks       bool
	ErrPoster      bool

	// Values for input if error
	ValTitle       string
	ValDescription string
	ValLinks       string
	//ValPoster      bool
}

func (d dataAddMovie) isError() bool {
	return d.ErrTitle || d.ErrDescription || d.ErrLinks || d.ErrPoster
}

type dataAccount struct {
	dataPageBase

	CurrentVotes   []*Movie
	TotalVotes     int
	AvailableVotes int

	SuccessMessage string

	PassError   []string
	NotifyError []string
	EmailError  []string

	ErrCurrentPass bool
	ErrNewPass     bool
	ErrEmail       bool
}

func (a dataAccount) IsErrored() bool {
	return a.ErrCurrentPass || a.ErrNewPass || a.ErrEmail
}

type dataNewAccount struct {
	dataPageBase

	ErrorMessage []string
	ErrName      bool
	ErrPass      bool
	ErrEmail     bool

	ValName           string
	ValEmail          string
	ValNotifyEnd      bool
	ValNotifySelected bool
}

type dataError struct {
	dataPageBase

	Message string
	Code    int
}
