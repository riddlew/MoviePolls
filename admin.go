package moviepoll

import (
	"fmt"
	"net/http"
	"strconv"
)

type dataAdminHome struct {
	dataPageBase
}

type dataAdminConfig struct {
	dataPageBase

	ErrorMessage []string

	MaxUserVotes           int
	EntriesRequireApproval bool
	VotingOpen             bool

	ErrMaxUserVotes bool
}

func (d dataAdminConfig) IsErrored() bool {
	return d.ErrMaxUserVotes
}

type dataAdminUsers struct {
	dataPageBase

	Users []*User
}

type dataAdminUserEdit struct {
	dataPageBase

	User           *User
	CurrentVotes   []*Movie
	AvailableVotes int

	PassError   []string
	NotifyError []string
}

func (s *Server) checkAdminRights(w http.ResponseWriter, r *http.Request) bool {
	user := s.getSessionUser(w, r)

	ok := true
	if user == nil || user.Privilege < PRIV_MOD {
		ok = false
	}

	if !ok {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
			return false
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return false
	}

	return true
}

func (s *Server) handlerAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	data := dataAdminHome{
		dataPageBase: s.newPageBase("Admin", w, r),
	}

	if err := s.executeTemplate(w, "adminHome", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	ulist, err := s.data.GetUsers(0, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Error getting users: %v", err),
			w, r)
		return
	}

	data := dataAdminUsers{
		dataPageBase: s.newPageBase("Admin - Users", w, r),
		Users:        ulist,
	}

	if err := s.executeTemplate(w, "adminUsers", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var uid int
	_, err := fmt.Sscanf(r.URL.Path, "/admin/user/%d", &uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Unable to parse user ID: %v", err),
			w, r)
		return
	}

	user, err := s.data.GetUser(uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get user: %v", err),
			w, r)
		return
	}

	config, err := s.data.GetConfig()
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get config: %v", err),
			w, r)
		return
	}

	totalVotes, err := config.GetInt("MaxUserVotes")
	if err != nil {
		fmt.Printf("Error getting MaxUserVotes config setting: %v\n", err)
		totalVotes = 5 // FIXME: define a default somewhere?
	}

	data := dataAdminUserEdit{
		dataPageBase: s.newPageBase("Admin - User Edit", w, r),

		User:         user,
		CurrentVotes: s.data.GetUserVotes(uid),
	}
	data.AvailableVotes = totalVotes - len(data.CurrentVotes)

	if r.Method == "POST" {
		// do a thing
	}

	if err := s.executeTemplate(w, "adminUserEdit", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminConfig(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	config, err := s.data.GetConfig()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get config values: %v", err),
			w, r)
		return
	}

	data := dataAdminConfig{
		dataPageBase: s.newPageBase("Admin - Config", w, r),
		ErrorMessage: []string{},
	}

	if r.Method == "POST" {
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Unable to parse form: %v\n", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		maxVotesStr := r.PostFormValue("MaxUserVotes")
		maxVotes, err := strconv.ParseInt(maxVotesStr, 10, 32)
		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("MaxUserVotes invalid: %v", err))
			data.ErrMaxUserVotes = true
		} else {
			config.SetInt("MaxUserVotes", int(maxVotes))
		}

		appReqStr := r.PostFormValue("EntriesRequireApproval")
		if appReqStr != "" {
			config.SetInt("EntriesRequireApproval", int(maxVotes))
		}

		clearPass := r.PostFormValue("ClearPassSalt")
		if clearPass != "" {
			config.Delete("PassSalt")
		}

		clearCookies := r.PostFormValue("ClearCookies")
		if clearCookies != "" {
			config.Delete("SessionAuth")
			config.Delete("SessionEncrypt")
		}

		err = s.data.SaveConfig(config)
		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("Unable to save config: %v", err))
		}
	}

	data.MaxUserVotes, err = config.GetInt("MaxUserVotes")
	if err != nil {
		data.MaxUserVotes = 5 // FIXME: define defaults elsewhere
	}

	data.EntriesRequireApproval, err = config.GetBool("EntriesRequireApproval")
	if err != nil {
		data.EntriesRequireApproval = false
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
