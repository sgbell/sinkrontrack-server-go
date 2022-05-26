package playlist

import (
	"errors"
	"mimpidev/sinkrontrack-server/internal/storage"
	"mimpidev/sinkrontrack-server/pkg/userLogin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/objectbox/objectbox-go/objectbox"
)

var executeFindPlaylist func(conditions []objectbox.Condition) ([]*Playlist, error)
var executeSelectPlaylist func(p *Playlist) error
var executeDeletePlaylist func(p *Playlist) error
var executeUpdatePlaylist func(p *Playlist) error
var executeFindUser func(conditions []objectbox.Condition) ([]*User, error)
var executeSelectUser func(m *User) error
var executeUpdateTrack func(t *Track) error
var executeDeleteTrack func(t *Track) error

func (p *Playlist) Delete() error {
	return executeDeletePlaylist(p)
}

func (p *Playlist) Select() error {
	return executeSelectPlaylist(p)
}

func (p *Playlist) Find(conditions ...objectbox.Condition) ([]*Playlist, error) {
	return executeFindPlaylist(conditions)
}

func (m *User) Find(conditions ...objectbox.Condition) ([]*User, error) {
	return executeFindUser(conditions)
}

func (m *User) Select() error {
	return executeSelectUser(m)
}

func (p *Playlist) Update() error {
	return executeUpdatePlaylist(p)
}

func (t *Track) Update() error {
	return executeUpdateTrack(t)
}

func (t *Track) Delete() error {
	return executeDeleteTrack(t)
}

func TestPlaylistCopy(t *testing.T) {
	t.Run("Copy Details from 1 instance of struct to playlist Object", func(t *testing.T) {
		var destPlaylist Playlist
		var srcPlaylist storage.Playlist
		srcPlaylist.Id = 1
		srcPlaylist.Name = "Test playlist"
		srcPlaylist.Uuid = "2898da6e-b222-4227-8b7b-6bbc239705b0"
		srcPlaylist.ClientIdLock = "sad8976sdf87sdf"
		srcPlaylist.ClientLockExpires = 57
		srcPlaylist.Elapsed = 132
		tr := &storage.Track{Id: 1,
			Path:             "/mnt/sdb/Album1/Track1.mp3",
			Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
			ArtistName:       "Barry",
			SongName:         "Song1",
			AlbumName:        "Album1",
			AlbumTrackNumber: 1}
		srcPlaylist.Tracks = append(srcPlaylist.Tracks, tr)
		tr = &storage.Track{Id: 2,
			Path:             "/mnt/sdb/Album1/Track2.mp3",
			Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
			ArtistName:       "Barry",
			SongName:         "Song2",
			AlbumName:        "Album1",
			AlbumTrackNumber: 2}
		srcPlaylist.Tracks = append(srcPlaylist.Tracks, tr)

		destPlaylist.Copy(&srcPlaylist)

		if destPlaylist.Id != srcPlaylist.Id ||
			destPlaylist.Name != srcPlaylist.Name ||
			destPlaylist.Uuid != srcPlaylist.Uuid ||
			destPlaylist.ClientIdLock != srcPlaylist.ClientIdLock ||
			destPlaylist.ClientLockExpires != srcPlaylist.ClientLockExpires ||
			destPlaylist.Elapsed != srcPlaylist.Elapsed ||
			destPlaylist.Tracks[0].Uuid != srcPlaylist.Tracks[0].Uuid {
			t.Error("The Copy function failed")
		}
	})
}

func TestCreatePlaylist(t *testing.T) {
	t.Run("Invalid User Token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("POST", "/playlist", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		CreatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Invalid JSON", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		var data = `{"name":"Test"`
		request := httptest.NewRequest("POST", "/playlist", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		CreatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Mismatch JSON", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		var data = `{"owner":"Test"}`
		request := httptest.NewRequest("POST", "/playlist", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		CreatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Error Loading user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeAddPlaylist = func(m *User, p *Playlist) (*uint64, error) {
			return &[]uint64{1}[0], nil
		}

		executeSelectUser = func(m *User) error {
			return errors.New("Can't load the user")
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("POST", "/playlist", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		CreatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Fail to add new Playlist to user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeAddPlaylist = func(m *User, p *Playlist) (*uint64, error) {
			return nil, errors.New("Failed to add playlist")
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 2
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("POST", "/playlist", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		CreatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Add new Playlist to user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeAddPlaylist = func(m *User, p *Playlist) (*uint64, error) {
			return &[]uint64{1}[0], nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 2
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("POST", "/playlist", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		CreatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestUpdatePlaylist(t *testing.T) {
	t.Run("Invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com"}`
		request := httptest.NewRequest("PATCH", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("no playlist in url", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("PATCH", "/playlist/", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("playlist uuid in url is invalid", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("PATCH", "/playlist/d9g87sdgf98-sdf98sdf9sdf", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("ClientLockExpires has not yet passed (ie Playlist is still locked)", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			var playlists []*Playlist
			p := &Playlist{}
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.ClientIdLock = "sd7fsd8f76sdf876sdf"
			p.ClientLockExpires = time.Now().Unix() - 1000*60
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			playlists = append(playlists, p)

			return playlists, nil
		}

		executeUpdatePlaylist = func(p *Playlist) error {
			return nil
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("PATCH", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusLocked {
			t.Errorf("Want status '%d', got '%d'", http.StatusLocked, responseRecorder.Code)
		}

	})
	t.Run("Incoming data is invalid", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			var playlists []*Playlist
			p := &Playlist{}
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			playlists = append(playlists, p)

			return playlists, nil
		}

		executeUpdatePlaylist = func(p *Playlist) error {
			return nil
		}

		var data = `{"name":"Test"`
		request := httptest.NewRequest("PATCH", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}

	})
	t.Run("Update Paylist throws an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			var playlists []*Playlist
			p := &Playlist{}
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			playlists = append(playlists, p)

			return playlists, nil
		}

		executeUpdatePlaylist = func(p *Playlist) error {
			return errors.New("Just throwing a test Error")
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("PATCH", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}

	})
	t.Run("Update Playlist details", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			var playlists []*Playlist
			p := &Playlist{}
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			playlists = append(playlists, p)

			return playlists, nil
		}

		executeUpdatePlaylist = func(p *Playlist) error {
			return nil
		}

		var data = `{"name":"Test"}`
		request := httptest.NewRequest("PATCH", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		UpdatePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestGetPlaylistByUrl(t *testing.T) {
	t.Run("Get Playlist from url", func(t *testing.T) {
		claims := &userLogin.Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			var playlists []*Playlist
			p := &Playlist{}
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			playlists = append(playlists, p)

			return playlists, nil
		}

		playlist, _, statusCode := getPlaylistByUrl("/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", claims)
		if *statusCode != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, statusCode)
		}

		if playlist == nil {
			t.Errorf("Expected playlist, but got nil")
		}
	})
	t.Run("Get Playlist from url as Admin user", func(t *testing.T) {
		claims := &userLogin.Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		isAdminUserVar = func(username string) bool {
			return true
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			var playlists []*Playlist
			p := &Playlist{}
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			playlists = append(playlists, p)

			return playlists, nil
		}

		playlist, _, statusCode := getPlaylistByUrl("/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", claims)
		if *statusCode != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, statusCode)
		}

		if playlist == nil {
			t.Errorf("Expected playlist, but got nil")
		}
	})
	t.Run("Invalid UUid", func(t *testing.T) {
		claims := &userLogin.Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		playlist, err, statusCode := getPlaylistByUrl("/playlist/1", claims)
		if *statusCode != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, statusCode)
		}

		if err == nil {
			t.Errorf("Expected error statement, but got nil")
		}

		if playlist != nil {
			t.Errorf("Expected nil playlist, but got a record")
		}
	})
	t.Run("Get Playlist from storage throws an error", func(t *testing.T) {
		claims := &userLogin.Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			return nil, errors.New("Error getting playlist")
		}

		playlist, err, statusCode := getPlaylistByUrl("/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", claims)
		if *statusCode != http.StatusInternalServerError {
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, statusCode)
		}

		if err == nil {
			t.Errorf("Expected error statement, but got nil")
		}

		if playlist != nil {
			t.Errorf("Expected nil playlist, but got a record")
		}
	})
	t.Run("Get Playlist from storage throws an error as admin user", func(t *testing.T) {
		claims := &userLogin.Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		isAdminUserVar = func(username string) bool {
			return true
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			return nil, errors.New("Error getting playlist")
		}

		playlist, err, statusCode := getPlaylistByUrl("/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", claims)
		if *statusCode != http.StatusInternalServerError {
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, statusCode)
		}

		if err == nil {
			t.Errorf("Expected error statement, but got nil")
		}

		if playlist != nil {
			t.Errorf("Expected nil playlist, but got a record")
		}
	})
	t.Run("Playlist does not exist", func(t *testing.T) {
		claims := &userLogin.Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		isAdminUserVar = func(username string) bool {
			return false
		}

		executeFindPlaylist = func(conditions []objectbox.Condition) ([]*Playlist, error) {
			return []*Playlist{}, nil
		}

		playlist, err, statusCode := getPlaylistByUrl("/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", claims)
		if *statusCode != http.StatusNotFound {
			t.Errorf("Want status '%d', got '%d'", http.StatusNotFound, statusCode)
		}

		if err == nil {
			t.Errorf("Expected error statement, but got nil")
		}

		if playlist != nil {
			t.Errorf("Expected nil playlist, but got a record")
		}
	})
}

func TestGetPlaylist(t *testing.T) {
	t.Run("Invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		request := httptest.NewRequest("GET", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		GetPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Successfully Get playlist", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			var p Playlist
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			return &p, nil, &[]int{http.StatusOK}[0]
		}
		request := httptest.NewRequest("GET", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		GetPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
	t.Run("getPlaylistByUrl throws an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			return nil, errors.New("Error Thrown"), &[]int{http.StatusInternalServerError}[0]
		}
		request := httptest.NewRequest("GET", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		GetPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
}

func TestDeletePlaylist(t *testing.T) {
	t.Run("Invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		request := httptest.NewRequest("GET", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		DeletePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Delete Playlist details", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			var p Playlist
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			return &p, nil, &[]int{http.StatusOK}[0]
		}

		executeDeletePlaylist = func(p *Playlist) error {
			return nil
		}

		request := httptest.NewRequest("DELETE", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		DeletePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
	t.Run("getPlaylistByUrl returns an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			return nil, errors.New("Error Thrown"), &[]int{http.StatusInternalServerError}[0]
		}

		request := httptest.NewRequest("DELETE", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		DeletePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
	t.Run("Delete Playlist details", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			var p Playlist
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a5",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a6",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				Uuid:             "48cf9b84-6162-430a-92ac-6804146ad2a7",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			return &p, nil, &[]int{http.StatusOK}[0]
		}

		executeDeletePlaylist = func(p *Playlist) error {
			return errors.New("Some kind of error")
		}

		request := httptest.NewRequest("DELETE", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4", nil)
		responseRecorder := httptest.NewRecorder()

		DeletePlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
}

func TestListPlaylist(t *testing.T) {
	t.Run("List Playlists", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com"
			m.Id = 2
			m.Enabled = true
			m.AdminUser = false
			p := &storage.Playlist{}
			p.Id = 1
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t := &storage.Track{Id: 1,
				Path:             "/mnt/sdb/Album1/Track1.mp3",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album1",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 2,
				Path:             "/mnt/sdb/Album1/Track2.mp3",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album1",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 3,
				Path:             "/mnt/sdb/Album1/Track3.mp3",
				ArtistName:       "Barry",
				SongName:         "Song3",
				AlbumName:        "Album1",
				AlbumTrackNumber: 3}
			p.Tracks = append(p.Tracks, t)
			m.Playlists = append(m.Playlists, p)
			p = &storage.Playlist{}
			p.Id = 2
			p.Name = "Test Playlist 2"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			t = &storage.Track{Id: 4,
				Path:             "/mnt/sdb/Album2/Track1.mp3",
				ArtistName:       "Barry",
				SongName:         "Song1",
				AlbumName:        "Album2",
				AlbumTrackNumber: 1}
			p.Tracks = append(p.Tracks, t)
			t = &storage.Track{Id: 5,
				Path:             "/mnt/sdb/Album2/Track2.mp3",
				ArtistName:       "Barry",
				SongName:         "Song2",
				AlbumName:        "Album2",
				AlbumTrackNumber: 2}
			p.Tracks = append(p.Tracks, t)
			m.Playlists = append(m.Playlists, p)

			return nil
		}

		request := httptest.NewRequest("GET", "/playlist/", nil)
		responseRecorder := httptest.NewRecorder()

		ListPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
	t.Run("List Users Playlists, but user has none", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com"
			m.Id = 2
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		request := httptest.NewRequest("GET", "/playlist/", nil)
		responseRecorder := httptest.NewRecorder()

		ListPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusNoContent {
			t.Errorf("Want status '%d', got '%d'", http.StatusNoContent, responseRecorder.Code)
		}
	})
	t.Run("invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com"
			m.Id = 2
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		request := httptest.NewRequest("GET", "/playlist/", nil)
		responseRecorder := httptest.NewRecorder()

		ListPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("user is not found", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			return userlist, nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com"
			m.Id = 2
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		request := httptest.NewRequest("GET", "/playlist/", nil)
		responseRecorder := httptest.NewRecorder()

		ListPlaylist(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
}

func TestAddTrack(t *testing.T) {
	t.Run("Invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		var data = `{"path":"/mnt/sdb/sorted-mp3z/Album/Track1","artistName":"Frankie","songName":"Track 1","albumName":"Album","albumTrackNumber":1}`
		request := httptest.NewRequest("POST", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4/track", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		AddTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("getPlaylist Failed", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			return nil, errors.New("Error Thrown"), &[]int{http.StatusInternalServerError}[0]
		}

		var data = `{"path":"/mnt/sdb/sorted-mp3z/Album/Track1","artistName":"Frankie","songName":"Track 1","albumName":"Album","albumTrackNumber":1}`
		request := httptest.NewRequest("POST", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4/track", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		AddTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
	t.Run("Unknown fields in JSON", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			var p Playlist
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			return &p, nil, &[]int{http.StatusOK}[0]
		}

		var data = `{"pathogen":"/mnt/sdb/sorted-mp3z/Album/Track1","artistName":"Frankie","songName":"Track 1","albumName":"Album","albumTrackNumber":1}`
		request := httptest.NewRequest("POST", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4/track", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		AddTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Adding a track Failed", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			var p Playlist
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			return &p, nil, &[]int{http.StatusOK}[0]
		}
		executeAddTrack = func(p *Playlist, t *Track) (*uint64, error) {
			return nil, errors.New("Could not save track")
		}

		var data = `{"path":"/mnt/sdb/sorted-mp3z/Album/Track1","artistName":"Frankie","songName":"Track 1","albumName":"Album","albumTrackNumber":1}`
		request := httptest.NewRequest("POST", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4/track", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		AddTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Adding a track Succeeded", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getPlaylistByUrlPath = func(url string, claims *userLogin.Claims) (*Playlist, error, *int) {
			var p Playlist
			p.Id = 1
			p.Uuid = "48cf9b84-6162-430a-92ac-6804146ad2a4"
			p.Name = "Test Playlist 1"
			p.CurrentTrackId = 1
			p.Elapsed = 0
			return &p, nil, &[]int{http.StatusOK}[0]
		}
		executeAddTrack = func(p *Playlist, t *Track) (*uint64, error) {
			return &[]uint64{1}[0], nil
		}

		var data = `{"path":"/mnt/sdb/sorted-mp3z/Album/Track1","artistName":"Frankie","songName":"Track 1","albumName":"Album","albumTrackNumber":1}`
		request := httptest.NewRequest("POST", "/playlist/48cf9b84-6162-430a-92ac-6804146ad2a4/track", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		AddTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestUpdateTrack(t *testing.T) {
	t.Run("Invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		var data = `{"path":"Album of Testing Awesomeness/01 - Track.ogg"}`
		request := httptest.NewRequest("PATCH", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Get track failed", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			return nil, errors.New("Error Thrown"), &[]int{http.StatusInternalServerError}[0]
		}

		var data = `{"path":"Album of Testing Awesomeness/01 - Track.ogg"}`
		request := httptest.NewRequest("PATCH", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
	t.Run("Unknown fields in JSON", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			var t Track
			t.Id = 1
			t.Uuid = "5e638c1c-adce-46de-b780-d8247bd91e78"
			t.ArtistName = "Test Artist"
			t.AlbumName = "Album of Testing Awesomeness"
			t.Path = "Test Artist/Album of Testing Awesomeness/01 - Track too awesome.mp3"
			t.SongName = "Track too awesome"
			t.AlbumTrackNumber = 1
			t.TrackLength = 320
			return &t, nil, &[]int{http.StatusOK}[0]
		}

		var data = `{"pathHome":"Album of Testing Awesomeness/01 - Track.ogg"}`
		request := httptest.NewRequest("PATCH", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Updating a track Failed", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			var t Track
			t.Id = 1
			t.Uuid = "5e638c1c-adce-46de-b780-d8247bd91e78"
			t.ArtistName = "Test Artist"
			t.AlbumName = "Album of Testing Awesomeness"
			t.Path = "Test Artist/Album of Testing Awesomeness/01 - Track too awesome.mp3"
			t.SongName = "Track too awesome"
			t.AlbumTrackNumber = 1
			t.TrackLength = 320
			return &t, nil, &[]int{http.StatusOK}[0]
		}

		executeUpdateTrack = func(t *Track) error {
			return errors.New("Error thrown")
		}

		var data = `{"path":"Album of Testing Awesomeness/01 - Track.ogg"}`
		request := httptest.NewRequest("PATCH", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Update a track Succeeded", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			var t Track
			t.Id = 1
			t.Uuid = "5e638c1c-adce-46de-b780-d8247bd91e78"
			t.ArtistName = "Test Artist"
			t.AlbumName = "Album of Testing Awesomeness"
			t.Path = "Test Artist/Album of Testing Awesomeness/01 - Track too awesome.mp3"
			t.SongName = "Track too awesome"
			t.AlbumTrackNumber = 1
			t.TrackLength = 320
			return &t, nil, &[]int{http.StatusOK}[0]
		}

		executeUpdateTrack = func(t *Track) error {
			return nil
		}

		var data = `{"path":"Album of Testing Awesomeness/01 - Track.ogg"}`
		request := httptest.NewRequest("PATCH", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestDeleteTrack(t *testing.T) {
	t.Run("Invalid token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			return nil, http.StatusUnauthorized
		}

		request := httptest.NewRequest("DELETE", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", nil)
		responseRecorder := httptest.NewRecorder()
		DeleteTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Get track failed", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			return nil, errors.New("Error Thrown"), &[]int{http.StatusInternalServerError}[0]
		}

		request := httptest.NewRequest("DELETE", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", nil)
		responseRecorder := httptest.NewRecorder()
		DeleteTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
	t.Run("Deleting a track Failed", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			var t Track
			t.Id = 1
			t.Uuid = "5e638c1c-adce-46de-b780-d8247bd91e78"
			t.ArtistName = "Test Artist"
			t.AlbumName = "Album of Testing Awesomeness"
			t.Path = "Test Artist/Album of Testing Awesomeness/01 - Track too awesome.mp3"
			t.SongName = "Track too awesome"
			t.AlbumTrackNumber = 1
			t.TrackLength = 320
			return &t, nil, &[]int{http.StatusOK}[0]
		}

		executeDeleteTrack = func(t *Track) error {
			return errors.New("Error thrown")
		}

		request := httptest.NewRequest("DELETE", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", nil)
		responseRecorder := httptest.NewRecorder()
		DeleteTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Deleting a track Succeeded", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*userLogin.Claims, int) {
			claims := &userLogin.Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		getTrackByUrlPath = func(path string, claims *userLogin.Claims) (*Track, error, *int) {
			var t Track
			t.Id = 1
			t.Uuid = "5e638c1c-adce-46de-b780-d8247bd91e78"
			t.ArtistName = "Test Artist"
			t.AlbumName = "Album of Testing Awesomeness"
			t.Path = "Test Artist/Album of Testing Awesomeness/01 - Track too awesome.mp3"
			t.SongName = "Track too awesome"
			t.AlbumTrackNumber = 1
			t.TrackLength = 320
			return &t, nil, &[]int{http.StatusOK}[0]
		}

		executeDeleteTrack = func(t *Track) error {
			return nil
		}

		request := httptest.NewRequest("DELETE", "/track/5e638c1c-adce-46de-b780-d8247bd91e78", nil)
		responseRecorder := httptest.NewRecorder()
		DeleteTrack(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Status Message: '%s'", responseRecorder.Body)
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}
