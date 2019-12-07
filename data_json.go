package moviepoll

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

//type jsonCycle
type jsonMovie struct {
	Id           int
	Name         string
	Links        []string
	Description  string
	CycleAddedId int
	Removed      bool
	Approved     bool
	Watched      *time.Time
	Poster       string
}

type jsonVote struct {
	UserId  int
	MovieId int
	CycleId int
}

type jsonConnector struct {
	filename     string `json:"-"`
	CurrentCycle int

	Cycles []*Cycle
	Movies []jsonMovie
	Users  []*User
	Votes  []jsonVote

	//Settings Configurator
	Settings configMap
}

func NewJsonConnector(filename string) (*jsonConnector, error) {
	if fileExists(filename) {
		return LoadJson(filename)
	}

	return &jsonConnector{
		filename:     filename,
		CurrentCycle: 0,
		Settings: configMap{
			"Active": configValue{CVT_BOOL, true},
		},
	}, nil
}

func LoadJson(filename string) (*jsonConnector, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	data := &jsonConnector{}
	err = json.Unmarshal(raw, data)
	if err != nil {
		return nil, fmt.Errorf("Unable to read JSON data: %v", err)
	}

	data.filename = filename

	return data, nil
}

func (j *jsonConnector) Save() error {
	raw, err := json.MarshalIndent(j, "", " ")
	if err != nil {
		return fmt.Errorf("Unable to marshal JSON data: %v", err)
	}

	err = ioutil.WriteFile(j.filename, raw, 0777)
	if err != nil {
		return fmt.Errorf("Unable to write JSON data: %v", err)
	}

	return nil
}

/*
   On determining the current cycle.

   Should the current cycle have an end date?
   If so, this would be the automatic end date for the cycle.
   If not, only the current cycle would have an end date, which would define
   the current cycle as the cycle without an end date.

   Otherwise, just store the current cycle's ID somewhere (current
   functionality).
*/
func (j *jsonConnector) GetCurrentCycle() *Cycle {
	for _, c := range j.Cycles {
		if j.CurrentCycle == c.Id {
			return c
		}
	}
	return nil
}

func (j *jsonConnector) AddCycle(end *time.Time) error {
	if j.Cycles == nil {
		j.Cycles = []*Cycle{}
	}

	c := &Cycle{Start: time.Now()}

	if end != nil {
		c.End = *end
		c.EndingSet = true
	} else {
		c.EndingSet = false
	}
	j.Cycles = append(j.Cycles, c)

	return j.Save()
}

func (j *jsonConnector) AddOldCycle(c *Cycle) error {
	if j.Cycles == nil {
		j.Cycles = []*Cycle{}
	}

	j.Cycles = append(j.Cycles, c)
	return j.Save()
}

func (j *jsonConnector) nextCycleId() int {
	highest := 0
	for _, c := range j.Cycles {
		if c.Id > highest {
			highest = c.Id
		}
	}
	return highest + 1
}

func (j *jsonConnector) AddMovie(movie *Movie) error {
	if j.Movies == nil {
		j.Movies = []jsonMovie{}
	}

	m := jsonMovie{
		Id:           movie.Id,
		Name:         movie.Name,
		Links:        movie.Links,
		Description:  movie.Description,
		CycleAddedId: movie.CycleAdded.Id,
		Removed:      movie.Removed,
		Approved:     movie.Approved,
		Poster:       movie.Poster,
	}

	j.Movies = append(j.Movies, m)

	return j.Save()
}

func (j *jsonConnector) GetMovie(id int) (*Movie, error) {
	movie := j.findMovie(id)
	if movie == nil {
		return nil, fmt.Errorf("Movie with ID %d not found.", id)
	}

	movie.Votes = j.findVotes(movie)
	return movie, nil
}

func (j *jsonConnector) GetActiveMovies() []*Movie {
	movies := []*Movie{}

	for _, m := range j.Movies {
		mov, _ := j.GetMovie(m.Id)
		if mov != nil && m.Watched == nil {
			movies = append(movies, mov)
		}
	}

	return movies
}

func (j *jsonConnector) GetPastCycles(start, end int) []*Cycle {
	// TODO: implement this
	return []*Cycle{}
}

func (j *jsonConnector) GetUser(userId int) (*User, error) {
	u := j.findUser(userId)
	if u == nil {
		return nil, fmt.Errorf("User not found with ID %s", userId)
	}
	return u, nil
}

func (j *jsonConnector) AddUser(user *User) error {
	for _, u := range j.Users {
		if u.Id == user.Id {
			return fmt.Errorf("User already exists with ID %d", user.Id)
		}
	}

	j.Users = append(j.Users, user)
	return j.Save()
}

func (j *jsonConnector) AddVote(userId, movieId, cycleId int) error {
	user := j.findUser(userId)
	if user == nil {
		return fmt.Errorf("User not found with ID %d", userId)
	}

	movie := j.findMovie(movieId)
	if movie == nil {
		return fmt.Errorf("Movie not found with ID %d", movieId)
	}

	cycle := j.findCycle(cycleId)
	if cycle == nil {
		return fmt.Errorf("Cycle not found with ID %d", cycleId)
	}

	j.Votes = append(j.Votes, jsonVote{userId, movieId, cycleId})
	return j.Save()
}

func (j *jsonConnector) findMovie(id int) *Movie {
	for _, m := range j.Movies {
		if m.Id == id {
			return &Movie{
				Id:          id,
				Name:        m.Name,
				Description: m.Description,
				Removed:     m.Removed,
				Approved:    m.Approved,
				CycleAdded:  j.findCycle(m.CycleAddedId),
				Links:       m.Links,
				Poster:      m.Poster,
			}
		}
	}

	return nil
}

func (j *jsonConnector) findCycle(id int) *Cycle {
	for _, c := range j.Cycles {
		if c.Id == id {
			return c
		}
	}
	return nil
}

func (j *jsonConnector) findVotes(movie *Movie) []*Vote {
	votes := []*Vote{}
	for _, v := range j.Votes {
		if v.MovieId == movie.Id {
			votes = append(votes, &Vote{
				Movie:      movie,
				CycleAdded: j.findCycle(v.CycleId),
				User:       j.findUser(v.UserId),
			})
		}
	}

	return votes
}

func (j *jsonConnector) findUser(id int) *User {
	for _, u := range j.Users {
		if u.Id == id {
			return u
		}
	}
	return nil
}

func (j *jsonConnector) GetConfig() (Configurator, error) {
	if j.Settings == nil {
		return configMap{}, nil
	}
	return j.Settings, nil
}

func (j *jsonConnector) SaveConfig(config Configurator) error {
	j.Settings = config.(configMap)
	return j.Save()
}
