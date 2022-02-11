package database

import (
	"fmt"
	//"os"
	"testing"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

var (
	conn TestableDataConnector

	testUser  *common.User
	testMovie *common.Movie
	testCycle *common.Cycle

	userId  int
	cycleId int
	movieId int

	userFail  bool
	movieFail bool
	cycleFail bool
)

func Test_AddCycle(t *testing.T) {
	future := time.Now().Local().AddDate(0, 0, 7)
	cycleId, err = conn.AddCycle(&future)
	if err != nil {
		t.Fatal(err)
	}

	if cycleId < 1 {
		t.Fatal("Invalid cycle Id returned")
	}

	t.Logf("created cycle Id: %d", cycleId)
}

func Test_GetCurrentCycle(t *testing.T) {
	if cycleId < 1 {
		t.Skip("Skipping due to previous failure")
	}

	cycle, err := conn.GetCurrentCycle()
	if err != nil {
		t.Fatal(err)
	}

	if cycle == nil {
		t.Fatal("GetCurrentCycle() returned nil cycle and nil error")
	}

	if cycle.Id != cycleId {
		t.Fatalf("GetCurrentCycle() returned the wrong cycle.  Id %d vs %d", cycle.Id, cycleId)
	}
	testCycle = cycle
}

func Test_AddMovie(t *testing.T) {
	if testCycle == nil {
		t.Skip("Skipping due to previous failure")
	}

	testDate := time.Now().Local()
	movieName := fmt.Sprintf("Test Movie %d", testDate.Unix())

	// Add Movie
	m := &common.Movie{
		Name: movieName,
		Links: []string{
			fmt.Sprintf("http://example.com/1/%d", testDate.Unix()),
			fmt.Sprintf("https://example.com/2/%d", testDate.Unix()),
		},
		Description: fmt.Sprintf("%s description", movieName),
		CycleAdded:  testCycle,
		Removed:     true,
		Approved:    false,
		Votes:       []*common.Vote{},
		Poster:      "unknown.jpg",
	}

	movieId, err = conn.AddMovie(m)
	if err != nil {
		movieFail = true
		t.Fatal(err)
	}

	if movieId < 1 {
		movieFail = true
		t.Fatal("Invalid movie Id returned")
	}

	testMovie = m
}

func Test_GetMovie(t *testing.T) {
	if testMovie == nil {
		t.Skip("Skipping due to previous failure")
	}

	movie, err := conn.GetMovie(movieId)
	if err != nil {
		t.Fatal(err)
	}

	if movie == nil {
		t.Fatal("GetMovie() returned nil movie and nil error")
	}

	compareMovies(testMovie, movie, t)
}

func Test_GetActiveMovies(t *testing.T) {
	if testMovie == nil {
		t.Skip("Skipping due to previous failure")
	}

	active, err := conn.GetActiveMovies()
	if err != nil {
		t.Fatal(err)
	}

	var movie *common.Movie
	for _, mov := range active {
		if mov.Id == movieId {
			movie = mov
			break
		}
	}

	compareMovies(testMovie, movie, t)
}

func Test_CheckMovieExists_True(t *testing.T) {
	if testMovie == nil {
		t.Skip("Skipping due to previous failure")
	}

	if ok, err := conn.CheckMovieExists(testMovie.Name); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("CheckMovieExists() failed")
	}
}

func Test_CheckMovieExists_False(t *testing.T) {
	val, err := conn.CheckMovieExists("doesn't exist")
	if err != nil {
		t.Fatal(err)
	}

	if val {
		t.Fatal("non existent movie apparently exists")
	}
}

func Test_AddUser(t *testing.T) {
	passDate := time.Now().UTC().Truncate(time.Second)
	name := fmt.Sprintf("test_user_parts_%d", passDate.Unix())
	testUser = &common.User{
		Id:                  -1, // this should be ignored when adding.
		Name:                name,
		Password:            `"hashed" password`,
		OAuthToken:          fmt.Sprintf("%s token", name),
		Email:               fmt.Sprintf("%s@example.com", name),
		NotifyCycleEnd:      true,
		NotifyVoteSelection: true,
		Privilege:           common.PRIV_MOD,
		PassDate:            passDate,
		RateLimitOverride:   true,
	}

	uid, err := conn.AddUser(testUser)
	if err != nil {
		t.Fatal(err)
	}

	testUser.Id = uid
	if testUser.Id == -1 {
		t.Fatal("User Id not updated")
	}
}

func Test_GetUser(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Fatal("Skipping due to previous failure")
	}

	u, err := conn.GetUser(testUser.Id)
	if err != nil {
		t.Fatal(err)
	}

	if u == nil {
		t.Fatal("GetUser() returned a nil user and no error")
	}

	compareUsers(testUser, u, t)
}

func Test_CheckUserExists(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Skip("Skipping due to previous failure")
	}

	exist, err := conn.CheckUserExists(testUser.Name)
	if err != nil {
		t.Fatal(err)
	}

	if !exist {
		t.Fatal("User doesn't exist when they should")
	}
}

func Test_UserLogin(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Skip("Skipping due to previous failure")
	}

	u, err := conn.UserLogin(testUser.Name, testUser.Password)
	if err != nil {
		t.Fatal(err)
	}

	if u == nil {
		t.Fatal("GetUser() returned a nil user and no error")
	}

	compareUsers(testUser, u, t)
}

func Test_GetUsers(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Skip("Skipping due to previous failure")
	}

	lst, err := conn.GetUsers(0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(lst) == 0 {
		t.Fatal("GetUsers() returned no users and no error")
	}

	var u *common.User
	for _, user := range lst {
		if testUser.Id == user.Id {
			u = user
			break
		}
	}

	compareUsers(testUser, u, t)
}

func testMySql_GetUserVotes(t *testing.T) {
	t.Skip("Test Not implemented")
}

func testMySql_GetPastCycles(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_AddOldCycle(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_AddVote(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_UpdateUser(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_UpdateMovie(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_UpdateCycle(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_UserVotedForMovie(t *testing.T) {
	t.Skip("Test Not implemented")
}

func Test_CfgInt(t *testing.T) {
	testDate := time.Now().Unix()
	data := int(testDate)
	key := fmt.Sprintf("test int %d", testDate)

	err := conn.SetCfgInt(key, data)
	if err != nil {
		t.Fatal(err)
	}

	val, err := conn.GetCfgInt(key, -1)
	if err != nil {
		t.Fatal(err)
	}

	if data != val {
		t.Fatalf("cfg int mismatch: %d vs %d", data, val)
	}

	// Make sure int/string variants return an error
	_, err = conn.GetCfgString(key, "nope.jpg")
	if err == nil {
		t.Fatal("GetCfgString() did not return an error for int key")
	}

	_, err = conn.GetCfgBool(key, false)
	if err == nil {
		t.Fatal("GetCfgBool() did not return an error for int key")
	}

	err = conn.DeleteCfgKey(key)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_CfgBool(t *testing.T) {
	testDate := time.Now().Unix()
	data := true
	key := fmt.Sprintf("test bool %d", testDate)

	err := conn.SetCfgBool(key, data)
	if err != nil {
		t.Fatal(err)
	}

	val, err := conn.GetCfgBool(key, !data)
	if err != nil {
		t.Fatal(err)
	}

	if data != val {
		t.Fatalf("cfg bool mismatch: %t vs %t", data, val)
	}

	// Make sure int/string variants return an error
	_, err = conn.GetCfgString(key, "nope.jpg")
	if err == nil {
		t.Fatal("GetCfgString() did not return an error for bool key")
	}

	_, err = conn.GetCfgInt(key, -1)
	if err == nil {
		t.Fatal("GetCfgInt() did not return an error for bool key")
	}

	err = conn.DeleteCfgKey(key)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_CfgString(t *testing.T) {
	testDate := time.Now().Unix()
	data := "test data"
	key := fmt.Sprintf("test string %d", testDate)

	err := conn.SetCfgString(key, data)
	if err != nil {
		t.Fatal(err)
	}

	val, err := conn.GetCfgString(key, "nope.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if data != val {
		t.Fatalf("cfg string mismatch: %q vs %q", data, val)
	}

	// Make sure int/bool variants return an error
	_, err = conn.GetCfgBool(key, false)
	if err == nil {
		t.Fatal("GetCfgBool() did not return an error for string key")
	}

	_, err = conn.GetCfgInt(key, -1)
	if err == nil {
		t.Fatal("GetCfgInt() did not return an error for string key")
	}

	err = conn.DeleteCfgKey(key)
	if err != nil {
		t.Fatal(err)
	}
}

//  - Add 4 cycles w/ 2 movies each cycle, each having a vote.
//  - Close each cycle selecting a single movie to watch (leaving 1 movie added
//    each cycle w/ a vote and still active).
//  - Decay votes older than 2 cycles.
func TestJson_DecayVotes(t *testing.T) {
	now := time.Now().Local()
	uid, err := conn.AddUser(&common.User{Name: "Test User"})
	if err != nil {
		t.Fatal(err)
	}

	moviesDecay := []int{}
	moviesNoDecay := []int{}
	//var lastMovieId int = -1
	for i := 0; i < 4; i++ {
		end := now
		curr, err := conn.GetCurrentCycle()
		if err != nil {
			t.Fatal(err)
		}
		//fmt.Println("current cycle: ", curr)
		if curr != nil {

			// add two movies to the current cycle
			m1 := &common.Movie{
				Name:        fmt.Sprintf("Movie %d.a Selected", i),
				Links:       []string{"http://example.com/"},
				Description: "",
				CycleAdded:  curr,
				Removed:     false,
				Approved:    true,
				Votes:       []*common.Vote{},
				Poster:      "",
			}

			m1_id, err := conn.AddMovie(m1)
			if err != nil {
				movieFail = true
				t.Fatal(err)
			}
			moviesNoDecay = append(moviesNoDecay, m1_id)

			m2 := &common.Movie{
				Name:        fmt.Sprintf("Movie %d.b", i),
				Links:       []string{"http://example.com/"},
				Description: "",
				CycleAdded:  curr,
				Removed:     false,
				Approved:    true,
				Votes:       []*common.Vote{},
				Poster:      "",
			}

			m2_id, err := conn.AddMovie(m2)
			if err != nil {
				movieFail = true
				t.Fatal(err)
			}

			if i < 2 {
				moviesDecay = append(moviesDecay, m2_id)
			} else {
				moviesNoDecay = append(moviesNoDecay, m2_id)
			}

			// Vote for both movies
			err = conn.AddVote(uid, m1_id)
			if err != nil {
				t.Fatal(err)
			}

			err = conn.AddVote(uid, m2_id)
			if err != nil {
				t.Fatal(err)
			}

			// End the cycle
			m, err := conn.GetMovie(m1_id)
			if err != nil {
				t.Fatal(err)
			}
			m.CycleWatched = curr

			err = conn.UpdateMovie(m)
			if err != nil {
				t.Fatal(err)
			}

			curr.Watched = []*common.Movie{m}
			curr.Ended = &end

			err = conn.UpdateCycle(curr)
			if err != nil {
				t.Fatal(err)
			}
			//fmt.Println("")
		}

		// Start a new cycle
		_, err = conn.AddCycle(&end)
		if err != nil {
			t.Fatal(err)
		}

		// Setup cycle's end
		now = now.Add(time.Hour * 24 * (time.Duration(i) + 1))
	}

	// test actually starts here, lol
	before, err := conn.Test_GetUserVotes(uid)
	if err != nil {
		t.Fatal(err)
	}

	err = conn.DecayVotes(2)
	if err != nil {
		t.Fatal(err)
	}

	after, err := conn.Test_GetUserVotes(uid)
	if err != nil {
		t.Fatal(err)
	}

	if len(after) == 0 {
		t.Log("User has no votes")
		t.Fail()
	}

	voteIds := []int{}
	for _, v := range after {
		for _, m := range moviesDecay {
			if m == v.Movie.Id {
				t.Logf("Failed to decay vote with movie ID %d", m)
				t.Fail()
			}
		}

		voteIds = append(voteIds, v.Movie.Id)
	}

	for _, m := range moviesNoDecay {
		if !containsInt(t, voteIds, m) {
			t.Logf("Decayed wrong vote with movie ID %d", m)
			t.Fail()
		}
	}

	if t.Failed() {
		t.Log("moviesNoDecay ", moviesNoDecay)
		t.Log("moviesDecay ", moviesDecay)

		past, err := conn.GetPastCycles(0, 5)
		if err != nil {
			t.Fatal(err)
		}

		watched := []*common.Movie{}
		for _, p := range past {
			watched = append(watched, p.Watched...)
		}

		fmt.Println("Watched")
		for _, m := range watched {
			fmt.Printf("  [%d] %s\n", m.Id, m.Name)
		}

		fmt.Println("Before")
		for _, v := range before {
			fmt.Printf("  [%d] %s\n", v.Movie.Id, v.Movie.Name)
		}

		fmt.Println("After")
		for _, v := range after {
			fmt.Printf("  [%d] %s\n", v.Movie.Id, v.Movie.Name)
		}
	}
}

/*
	Cleanup
*/

func Test_DeleteVote(t *testing.T) {
	if testUser == nil || testUser.Id < 1 ||
		testMovie == nil || testMovie.Id < 1 ||
		testCycle == nil || testCycle.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	err := conn.DeleteVote(testUser.Id, testMovie.Id)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_DeleteMovie(t *testing.T) {
	if movieFail || testMovie == nil || testMovie.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	err := conn.DeleteMovie(testMovie.Id)
	if err != nil {
		t.Fatal(err)
	}
	testMovie = nil
}

func Test_DeleteCycle(t *testing.T) {
	if movieFail || cycleFail || testMovie != nil || testCycle == nil || testCycle.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	err := conn.DeleteCycle(testCycle.Id)
	if err != nil {
		t.Fatal(err)
	}
	testCycle = nil
}

func Test_DeleteUser(t *testing.T) {
	if userFail || testUser == nil || testUser.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	if err := conn.DeleteUser(testUser.Id); err != nil {
		t.Fatal(err)
	}
}
