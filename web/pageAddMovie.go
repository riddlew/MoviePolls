package web

import (
	"fmt"
	"net/http"

	"github.com/zorchenhimer/MoviePolls/logic"
)

func (s *webServer) handlerPageAddMovie(w http.ResponseWriter, r *http.Request) {

	// Get the user which adds a movie
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	// Get the current cycle to see if we can add a movie
	currentCycle, err := s.backend.GetCurrentCycle()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get current cycle: %v", err)
		return
	}

	if currentCycle == nil {
		s.doError(
			http.StatusInternalServerError,
			"No cycle active!",
			w, r)
		return
	}

	formfillEnabled, err := s.backend.GetFormFillEnabled()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", logic.ConfigFormfillEnabled, err)
		return
	}

	autofillEnabled, err := s.backend.GetAutofillEnabled()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to determine if autofill is enabled")
		return
	}

	maxTitleLen, err := s.backend.GetMaxTitleLength()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", logic.ConfigMaxTitleLength, err)
		return
	}

	maxDescriptionLen, err := s.backend.GetMaxDescriptionLength()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", logic.ConfigMaxDescriptionLength, err)
		return
	}

	maxLinkLen, err := s.backend.GetMaxLinkLength()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", logic.ConfigMaxLinkLength, err)
		return
	}

	maxRemLen, err := s.backend.GetMaxRemarksLength()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", logic.ConfigMaxRemarksLength, err)
		return
	}

	data := struct {
		dataPageBase

		// eg, "Title": InputField{}
		Fields map[string]*logic.InputField

		AutofillEnabled bool
		FormfillEnabled bool

		MaxTitleLength       int
		MaxDescriptionLength int
		MaxLinkLength        int
		MaxRemarksLength     int

		HasError  bool
		FileError error
	}{
		dataPageBase: s.newPageBase("Add Movie", w, r),

		AutofillEnabled: autofillEnabled,
		FormfillEnabled: formfillEnabled,

		MaxTitleLength:       maxTitleLen,
		MaxDescriptionLength: maxDescriptionLen,
		MaxLinkLength:        maxLinkLen,
		MaxRemarksLength:     maxRemLen,
	}

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4096)
		if err != nil {
			s.l.Error("Error parsing movie form: %v", err)
		}

		input := make(map[string]*logic.InputField)
		for key, slice := range r.MultipartForm.Value {
			if len(slice) == 0 {
				continue
			}
			input[key] = &logic.InputField{Value: slice[0]}
		}

		file, fileHeader, _ := r.FormFile("PosterFile")

		// if err is not nil, fields is not nil
		movieId, fields := s.backend.AddMovie(input, user, file, fileHeader)
		hasError := false
		for _, field := range fields {
			if field.Error != nil {
				hasError = true
			}
		}
		if !hasError || movieId != -1 {
			http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
			return
		} else {
			data.Fields = fields
			data.HasError = hasError
		}
	}
	if err := s.executeTemplate(w, "addmovie", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
