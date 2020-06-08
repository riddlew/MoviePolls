package moviepoll

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MoviePolls/common"
	mpd "github.com/zorchenhimer/MoviePolls/data"
)

const SessionName string = "moviepoll-session"

// defaults
const (
	DefaultMaxUserVotes           int    = 5
	DefaultEntriesRequireApproval bool   = false
	DefaultVotingEnabled          bool   = false
	DefaultTmdbToken              string = ""
)

// configuration keys
const (
	ConfigVotingEnabled          string = "VotingEnabled"
	ConfigMaxUserVotes           string = "MaxUserVotes"
	ConfigEntriesRequireApproval string = "EntriesRequireApproval"
	ConfigTmdbToken              string = "TmdbToken"
)

type Options struct {
	Listen   string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug    bool   // debug logging to console
	LogLevel common.LogLevel
	LogFile  string
}

type Server struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
	data      mpd.DataConnector

	cookies      *sessions.CookieStore
	passwordSalt string

	// For claiming the first admin account
	adminTokenUrl string
	adminTokenKey string

	l *common.Logger
}

func NewServer(options Options) (*Server, error) {
	if options.Listen == "" {
		options.Listen = ":8090"
	}

	l, err := common.NewLogger(options.LogLevel, options.LogFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to setup logger: %v", err)
	}

	data, err := mpd.GetDataConnector("json", "db/data.json")
	if err != nil {
		return nil, fmt.Errorf("Unable to load json data: %v", err)
	}

	hs := &http.Server{
		Addr: options.Listen,
	}

	authKey, err := data.GetCfgString("SessionAuth", "")
	if err != nil || authKey == "" {
		authKey = getCryptRandKey(64)
		data.SetCfgString("SessionAuth", authKey)
	}

	encryptKey, err := data.GetCfgString("SessionEncrypt", "")
	if err != nil || encryptKey == "" {
		encryptKey = getCryptRandKey(32)
		data.SetCfgString("SessionEncrypt", encryptKey)
	}

	if options.Debug {
		fmt.Println("Debug mode turned on")
	}

	server := &Server{
		debug: options.Debug,
		data:  data,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
		l:       l,
	}

	server.passwordSalt, err = server.data.GetCfgString("PassSalt", "")
	if err != nil || server.passwordSalt == "" {
		server.passwordSalt = getCryptRandKey(32)
		server.data.SetCfgString("PassSalt", server.passwordSalt)
	}

	adminExists, err := server.CheckAdminExists()
	if err != nil {
		return nil, err
	}

	if !adminExists {
		url, err := generatePass()
		if err != nil {
			return nil, fmt.Errorf("Error generating admin token URL: %v", err)
		}

		key, err := generatePass()
		if err != nil {
			return nil, fmt.Errorf("Error generating admin token key: %v", err)
		}

		server.adminTokenUrl = url
		server.adminTokenKey = key

		fmt.Printf("Claim admin: http://<host>/auth/%s Password: %s\n", url, key)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/movie/", server.handlerMovie)
	mux.HandleFunc("/static/", server.handlerStatic)
	mux.HandleFunc("/poster/", server.handlerPoster)
	mux.HandleFunc("/add", server.handlerAddMovie)

	// list of past cycles
	mux.HandleFunc("/history", server.handlerHistory)

	mux.HandleFunc("/user", server.handlerUser)
	mux.HandleFunc("/user/login", server.handlerUserLogin)
	mux.HandleFunc("/user/logout", server.handlerUserLogout)
	mux.HandleFunc("/user/new", server.handlerUserNew)

	mux.HandleFunc("/vote/", server.handlerVote)
	mux.HandleFunc("/", server.handlerRoot)
	mux.HandleFunc("/favicon.ico", server.handlerFavicon)

	mux.HandleFunc("/auth/", server.handlerAuth)
	mux.HandleFunc("/admin/", server.handlerAdmin)
	mux.HandleFunc("/admin/config", server.handlerAdminConfig)
	mux.HandleFunc("/admin/cycles", server.handlerAdminCycles)
	// mux.HandleFunc("/admin/nextcycle", server.handlerAdminNextCycle)
	mux.HandleFunc("/admin/user/", server.handlerAdminUserEdit)
	mux.HandleFunc("/admin/users", server.handlerAdminUsers)
	mux.HandleFunc("/admin/movies", server.handlerAdminMovies)

	hs.Handler = mux
	server.s = hs

	err = server.registerTemplates()
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (server *Server) Run() error {
	if server.debug {
		fmt.Printf("Listening on address %s\n", server.s.Addr)
	}
	return server.s.ListenAndServe()
}

func (s *Server) CheckAdminExists() (bool, error) {
	found, end := false, false

	start := 0
	count := 20
	for !found && !end {
		users, err := s.data.GetUsers(start, 20)
		if err != nil {
			return false, err
		}
		start += count

		if err != nil {
			return false, nil
		}

		if len(users) == 0 {
			return false, nil
		}

		for _, u := range users {
			if u.IsAdmin() {
				return true, nil
			}
		}
	}

	fmt.Println("[CheckAdminExists] end of loop")
	return false, nil
}

func (s *Server) AddUser(user *common.User) error {
	user.Password = s.hashPassword(user.Password)
	_, err := s.data.AddUser(user)
	return err
}

func (s *Server) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	if common.FileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handlerStatic(w http.ResponseWriter, r *http.Request) {
	file := "static/" + filepath.Base(r.URL.Path)
	if s.debug {
		fmt.Printf("Attempting to serve file %q\n", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerPoster(w http.ResponseWriter, r *http.Request) {
	file := "posters/" + filepath.Base(r.URL.Path)
	if s.debug {
		fmt.Printf("Attempting to serve file %q\n", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerAddMovie(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	current, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		fmt.Printf("Unable to get current cycle: %v\n", err)
		return
	}

	if current == nil {
		s.doError(
			http.StatusInternalServerError,
			"No cycle active!",
			w, r)
		return
	}

	data := dataAddMovie{
		dataPageBase: s.newPageBase("Add Movie", w, r),
	}

	err = r.ParseMultipartForm(4096)
	if err != nil {
		fmt.Printf("Error parsing movie form: %v\n", err)
	}

	if r.Method == "POST" {
		errText := []string{}

		linktext := strings.ReplaceAll(r.FormValue("Links"), "\r", "")
		data.ValLinks = linktext

		links := strings.Split(linktext, "\n")
		links, err = common.VerifyLinks(links)
		if err != nil {
			fmt.Printf("bad link: %v\n", err)
			data.ErrLinks = true
			errText = append(errText, "Invalid link(s) given.")
		}

		data.ValTitle = strings.TrimSpace(r.FormValue("MovieName"))
		movieExists, err := s.data.CheckMovieExists(r.FormValue("MovieName"))
		if err != nil {
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to check if movie exists: %v", err),
				w, r)
			return
		}

		if movieExists {
			data.ErrTitle = true
			fmt.Println("Movie exists")
			errText = append(errText, "Movie already exists")
		}

		if data.ValTitle == "" && !(r.FormValue("AutofillBox") == "on") {
			errText = append(errText, "Missing movie title")
			data.ErrTitle = true
		}

		descr := strings.TrimSpace(r.FormValue("Description"))
		data.ValDescription = descr
		if len(descr) == 0 && !(r.FormValue("AutofillBox") == "on") {
			data.ErrDescription = true
			errText = append(errText, "Missing description")
		}

		movie := &common.Movie{
			Name:        strings.TrimSpace(r.FormValue("MovieName")),
			Description: strings.TrimSpace(r.FormValue("Description")),
			Votes:       []*common.Vote{},
			Links:       links,
			Poster:      "data/unknown.jpg", // 165x250
		}

		if r.FormValue("AutofillBox") == "on" {
			results, errors, rerenderSite := s.handleAutofill(links, w, r)

			if len(errors) > 0 {
				errText = append(errText, errors...)
				data.ErrAutofill = true

				if rerenderSite {
					data.ErrorMessage = errText
					if err := s.executeTemplate(w, "addmovie", data); err != nil {
						fmt.Printf("Error rendering template: %v\n", err)
					}
					return
				}
			} else {
				movie.Name = results[0]
				movie.Description = results[1]
				movie.Poster = results[2]
			}

		} else {
			movie.Name = strings.TrimSpace(r.FormValue("MovieName"))
			movie.Description = strings.TrimSpace(r.FormValue("Description"))

			posterFileName := strings.TrimSpace(r.FormValue("MovieName"))

			posterFile, _, _ := r.FormFile("PosterFile")

			if posterFile != nil {
				file, err := uploadFile(r, posterFileName)

				if err != nil {
					data.ErrPoster = true
					errText = append(errText, err.Error())
				} else {
					movie.Poster = file
				}
			}
		}
		var movieId int

		if !data.isError() {
			movie.AddedBy = user
			movieId, err = s.data.AddMovie(movie)
		}

		if err == nil && !data.isError() {
			http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
			return
		}

		//data.ErrorMessage = strings.Join(errText, "<br />")
		data.ErrorMessage = errText
		fmt.Printf("Movie not added. isError(): %t\nerr: %v\n", data.isError(), err)
	}

	if err := s.executeTemplate(w, "addmovie", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) doError(code int, message string, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%d for %q\n", code, r.URL.Path)
	dataErr := dataError{
		dataPageBase: s.newPageBase("Error", w, r),
		Message:      message,
		Code:         code,
	}

	w.WriteHeader(http.StatusNotFound)
	if err := s.executeTemplate(w, "error", dataErr); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	movieList := []*common.Movie{}

	if r.Body != http.NoBody {
		err := r.ParseForm()
		if err != nil {
			fmt.Printf("[ERR] %v\n", err)
		}
		searchVal := r.FormValue("search")

		movieList, err = s.data.SearchMovieTitles(searchVal)
		if err != nil {
			fmt.Printf("[ERR] %v\n", err)
		}
	} else {
		var err error = nil
		movieList, err = s.data.GetActiveMovies()
		if err != nil {
			fmt.Printf("[ERR] %v\n", err)
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get active movies. Please contact the server admin."),
				w, r)
			return
		}
	}

	data := struct {
		dataPageBase
		Movies        []*common.Movie
		VotingEnabled bool
	}{
		dataPageBase: s.newPageBase("Current Cycle", w, r),
		Movies:       common.SortMoviesByName(movieList),
	}

	data.VotingEnabled, _ = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)

	if err := s.executeTemplate(w, "cyclevotes", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		//http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handlerMovie(w http.ResponseWriter, r *http.Request) {
	var movieId int
	var command string
	n, err := fmt.Sscanf(r.URL.String(), "/movie/%d/%s", &movieId, &command)
	if err != nil && n == 0 {
		dataError := dataMovieError{
			dataPageBase: s.newPageBase("Error", w, r),
			ErrorMessage: "Missing movie ID",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			fmt.Println(err)
		}
		return
	}

	movie, err := s.data.GetMovie(movieId)
	if err != nil {
		dataError := dataMovieError{
			dataPageBase: s.newPageBase("Error", w, r),
			ErrorMessage: "Movie not found",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			fmt.Println("movie not found: " + err.Error())
		}
		return
	}

	// fmt.Println(movie)

	data := dataMovieInfo{
		dataPageBase: s.newPageBase(movie.Name, w, r),
		Movie:        movie,
	}

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}

// outsourced autofill logic
func (s *Server) handleAutofill(links []string, w http.ResponseWriter, r *http.Request) ([]string, []string, bool) {
	// internal error log
	errors := []string{}
	// bool to check if the site should be rerendered
	rerenderSite := false
	// slice for the api results
	var results []string
	// make sure we have a link to look at
	if len(links) >= 1 {
		sourcelink := links[0]

		if strings.Contains(sourcelink, "myanimelist") {
			// Get Data from MAL (jikan api)
			rgx := regexp.MustCompile(`[htp]{4}s?:\/\/[^\/]*\/anime\/([0-9]*)`)
			match := rgx.FindStringSubmatch(sourcelink)
			id := match[1]

			sourceAPI := jikan{id: id}

			// Return early when the title already exists
			title, err := sourceAPI.getTitle()
			if err == nil {
				exists, _ := s.data.CheckMovieExists(title)
				if err == nil {
					if exists {
						errors = append(errors, "Movie already exists")
						rerenderSite = true
						return nil, errors, rerenderSite
					}
				}
			}

			results, err = getMovieData(sourceAPI)

			if err != nil {
				// error while getting data from the api
				errors = append(errors, err.Error())
			}

		} else if strings.Contains(sourcelink, "imdb") {
			// Retrieve token from database
			token, err := s.data.GetCfgString("TmdbToken", "")
			if err != nil || token == "" {
				errors = append(errors, "TmdbToken is either empty or not set in the admin config")
				rerenderSite = true
				return nil, errors, rerenderSite
			}

			// get the movie id
			rgx := regexp.MustCompile(`[htp]{4}s?:\/\/[^\/]*\/title\/(tt[0-9]*)`)
			match := rgx.FindStringSubmatch(sourcelink)
			id := match[1]

			sourceAPI := tmdb{id: id, token: token}

			// Return early when the title already exists
			title, err := sourceAPI.getTitle()
			if err == nil {
				exists, _ := s.data.CheckMovieExists(title)
				if err == nil {
					if exists {
						errors = append(errors, "Movie already exists")
						rerenderSite = true
						return nil, errors, rerenderSite
					}
				}

				results, err = getMovieData(sourceAPI)

				if err != nil {
					// errors from getMovieData
					errors = append(errors, err.Error())
				}

			} else {
				// Errors from sourceAPI.getTitle
				errors = append(errors, err.Error())
			}
		} else {
			// neither IMDB nor MAL link
			errors = append(errors, "To use autofill use an imdb or myanimelist link as first link")
		}

		if len(results) > 0 {
			if results[0] == "" && results[1] == "" && results[2] == "" {
				errors = append(errors, "The provided imdb link is not a link to a movie!")
			}

			//duplicate check
			movieExists, err := s.data.CheckMovieExists(results[0])
			if err != nil {
				s.doError(
					http.StatusInternalServerError,
					fmt.Sprintf("Unable to check if movie exists: %v", err),
					w, r)
				return nil, errors, rerenderSite
			}

			if movieExists {
				errors = append(errors, "Movie already exists")
			}
		} else {
			errors = append(errors, "No results retrived from API")
		}
	} else {
		errors = append(errors, "No links provided")
	}
	return results, errors, rerenderSite
}

func uploadFile(r *http.Request, name string) (filepath string, err error) {
	// 10 MB upload limit
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("PosterFile")

	if err != nil {
		fmt.Printf("%v\n", err.Error())
		return "", fmt.Errorf("Unalbe to retrive the file")
	}

	defer file.Close()

	fmt.Printf("Uploaded File: %v - Size %v\n", handler.Filename, handler.Size)

	tempFile, err := ioutil.TempFile("posters", name+"-*.png")

	if err != nil {
		return "", fmt.Errorf("Error while saving file to disk\n%v\n", err)
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)

	if err != nil {
		return "", err
	}

	tempFile.Write(fileBytes)

	fmt.Printf("Filename: %v\n", tempFile.Name())

	return tempFile.Name(), nil
}

// List of past cycles
func (s *Server) handlerHistory(w http.ResponseWriter, r *http.Request) {
	past, err := s.data.GetPastCycles(0, 10)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("something went wrong :C"),
			w, r)
		fmt.Println("Unable to get past cycles: ", err)
		return
	}

	data := struct {
		dataPageBase
		Cycles []*common.Cycle
	}{
		dataPageBase: s.newPageBase("Cycle History", w, r),
		Cycles:       past,
	}

	if err := s.executeTemplate(w, "history", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
