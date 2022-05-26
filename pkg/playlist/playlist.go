package playlist

import (
	"encoding/json"
	"errors"
	"mimpidev/sinkrontrack-server/internal/storage"
	"mimpidev/sinkrontrack-server/internal/webhelper"
	"mimpidev/sinkrontrack-server/pkg/userLogin"
	"net/http"
	"time"
)

type Playlist struct {
	storage.Playlist
}

type User struct {
	storage.User
}

type Track struct {
	storage.Track
}

type TrackData struct {
	Path             string `json:"path,omitempty"`
	ArtistName       string `json:"artistName,omitempty"`
	SongName         string `json:"songName,omitempty"`
	AlbumName        string `json:"albumName,omitempty"`
	AlbumTrackNumber int    `json:"albumTrackNumber,omitempty"`
}

type UpdatePlaylistData struct {
	Name           string `json:"name,omitempty"`
	CurrentTrackId uint16 `json:"currentTrack,omitempty"`
	Elapsed        *int   `json:"elapsed,omitempty"`
}

type PlaylistData struct {
	Name string `json:"name,omitempty"`
}

var checkTokenVar = userLogin.CheckToken
var isAdminUserVar = storage.IsAdminUser
var getPlaylistByUrlPath = getPlaylistByUrl
var getTrackByUrlPath = getTrackByUrl
var copyPlaylist = deepCopyPlaylist

func (p *Playlist) Copy(src *storage.Playlist) {
	copyPlaylist(src, p)
}

func deepCopyPlaylist(src *storage.Playlist, dest *Playlist) {
	dest.Id = src.Id
	dest.Uuid = src.Uuid
	dest.Name = src.Name
	dest.CurrentTrackId = src.CurrentTrackId
	dest.Elapsed = src.Elapsed
	dest.ClientIdLock = src.ClientIdLock
	dest.ClientLockExpires = src.ClientLockExpires
	for _, sTrack := range src.Tracks {
		dest.Tracks = append(dest.Tracks, sTrack)
	}
}

var executeAddPlaylist = func(m *User, p *Playlist) (*uint64, error) {
	var user *storage.User
	storage.DeepCopy(m, user)
	var playlist *storage.Playlist
	storage.DeepCopy(p, playlist)
	return storage.UserAddPlaylist(user, playlist)
}

var executeAddTrack = func(p *Playlist, t *Track) (*uint64, error) {
	var playlist *storage.Playlist
	storage.DeepCopy(p, playlist)
	var track *storage.Track
	storage.DeepCopy(t, track)
	return storage.PlaylistAddTrack(playlist, track)
}

func (m *User) AddPlaylist(p *Playlist) (*uint64, error) {
	return executeAddPlaylist(m, p)
}

func (p *Playlist) AddTrack(t *Track) (*uint64, error) {
	return executeAddTrack(p, t)
}

func CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	var playlistData PlaylistData
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&playlistData)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	var playlist *Playlist
	playlist = &Playlist{}
	storage.DeepCopy(playlistData, playlist)

	// Load User from claims
	var user *User
	user = new(User)
	user.EmailAddress = claims.Username
	err = user.Select()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusUnauthorized}[0]) {
		return
	}
	_, err = user.AddPlaylist(playlist)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	json.NewEncoder(w).Encode(playlist)
	return

}

func UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	playlist, err, httpStatus := getPlaylistByUrlPath(r.URL.Path, claims)
	if err != nil {
		if webhelper.ReturnError(w, r, err, httpStatus) {
			return
		}
	}

	now := time.Now().Unix()
	if playlist.ClientLockExpires > 0 &&
		playlist.ClientLockExpires < now {
		webhelper.ReturnError(w, r, errors.New("Playlist is locked"), &[]int{http.StatusLocked}[0])
		return
	}

	var playlistData UpdatePlaylistData
	err = json.NewDecoder(r.Body).Decode(&playlistData)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	playlist.ClientIdLock = claims.Id
	expires := time.Now().Add(time.Minute * 10)
	playlist.ClientLockExpires = expires.Unix()

	if playlistData.Name != "" {
		playlist.Name = playlistData.Name
	}
	if playlistData.CurrentTrackId != 0 {
		playlist.CurrentTrackId = playlistData.CurrentTrackId
	}
	if playlistData.Elapsed != nil {
		playlist.Elapsed = *playlistData.Elapsed
	}

	err = playlist.Update()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	var returnPlaylist Playlist
	returnPlaylist.Uuid = playlist.Uuid
	returnPlaylist.Name = playlist.Name
	returnPlaylist.CurrentTrackId = playlist.CurrentTrackId
	returnPlaylist.Elapsed = playlist.Elapsed
	returnPlaylist.Tracks = append(returnPlaylist.Tracks, playlist.Tracks...)

	json.NewEncoder(w).Encode(returnPlaylist)
	return
}

func getTrackByUrl(path string, claims *userLogin.Claims) (*Track, error, *int) {
	uuid, err := webhelper.GetUUidFromUrl(path, "^/tracks/([^/]+)$")
	if err != nil {
		return nil, err, &[]int{http.StatusBadRequest}[0]
	}

	var track Track
	var tracks []*Track

	if !isAdminUserVar(claims.Username) {
		trackResults, err1 := track.Find(storage.Track_.Uuid.Equals(*uuid, true),
			storage.User_.Playlists.Link(storage.User_.EmailAddress.Equals(claims.Username, true)))
		err = err1
		for _, st := range trackResults {
			t := &Track{}
			storage.DeepCopy(st, t)
			tracks = append(tracks, t)
		}
	} else {
		trackResults, err1 := track.Find(storage.Track_.Uuid.Equals(*uuid, true))
		err = err1
		for _, st := range trackResults {
			t := &Track{}
			storage.DeepCopy(st, t)
			tracks = append(tracks, t)
		}
	}
	if err != nil {
		return nil, err, &[]int{http.StatusInternalServerError}[0]
	}
	if len(tracks) != 1 {
		err := errors.New("Tracks is invalid")
		return nil, err, &[]int{http.StatusNotFound}[0]
	}

	return tracks[0], nil, &[]int{http.StatusOK}[0]
}

func getPlaylistByUrl(path string, claims *userLogin.Claims) (*Playlist, error, *int) {
	uuid, err := webhelper.GetUUidFromUrl(path, "^/playlist/([^/]+)$")
	if err != nil {
		return nil, err, &[]int{http.StatusBadRequest}[0]
	}

	var playlist Playlist
	var playlists []*Playlist

	if !isAdminUserVar(claims.Username) {
		playlistResults, err1 := playlist.Find(storage.Playlist_.Uuid.Equals(*uuid, true),
			storage.User_.Playlists.Link(storage.User_.EmailAddress.Equals(claims.Username, true)))
		err = err1
		for _, sp := range playlistResults {
			p := &Playlist{}
			storage.DeepCopy(sp, p)
			playlists = append(playlists, p)
		}
	} else {
		playlistResults, err1 := playlist.Find(storage.Playlist_.Uuid.Equals(*uuid, true))
		err = err1
		for _, sp := range playlistResults {
			p := &Playlist{}
			storage.DeepCopy(sp, p)
			playlists = append(playlists, p)
		}
	}
	if err != nil {
		return nil, err, &[]int{http.StatusInternalServerError}[0]
	}
	if len(playlists) != 1 {
		err := errors.New("Playlist is invalid")
		return nil, err, &[]int{http.StatusNotFound}[0]
	}

	return playlists[0], nil, &[]int{http.StatusOK}[0]
}

func GetPlaylist(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	playlist, err, httpStatus := getPlaylistByUrlPath(r.URL.Path, claims)
	if err != nil {
		if webhelper.ReturnError(w, r, err, httpStatus) {
			return
		}
	}

	json.NewEncoder(w).Encode(playlist)
	return
}

func DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	playlist, err, statusCode := getPlaylistByUrlPath(r.URL.Path, claims)
	if err != nil {
		if webhelper.ReturnError(w, r, err, statusCode) {
			return
		}
	}

	err = playlist.Delete()
	if err != nil {
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusInternalServerError}[0]) {
			return
		}
	}
	var responseDetails webhelper.Response
	responseDetails.Message = "Record Successfully Deleted"
	json.NewEncoder(w).Encode(responseDetails)
	return
}

func ListPlaylist(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	// Load user, to pull playlists
	var user User
	userList, err := user.Find(storage.User_.EmailAddress.Equals(claims.Username, true))
	if err != nil ||
		len(userList) == 0 {
		err := errors.New("Failed to Find user account")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusUnauthorized}[0]) {
			return
		}
	}
	user.Id = userList[0].Id
	err = user.Select()
	if user.Playlists == nil {
		err := errors.New("User has 0 playlists")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusNoContent}[0]) {
			return
		}
	}
	json.NewEncoder(w).Encode(user.Playlists)
	return
}

func AddTrack(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}
	playlist, err, httpStatus := getPlaylistByUrlPath(r.URL.Path, claims)
	if err != nil {
		if webhelper.ReturnError(w, r, err, httpStatus) {
			return
		}
	}

	var trackData TrackData
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&trackData)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	var track *Track
	track = &Track{}
	storage.DeepCopy(trackData, track)
	_, err = playlist.AddTrack(track)

	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	json.NewEncoder(w).Encode(track)
	return
}

func UpdateTrack(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}
	track, err, httpStatus := getTrackByUrlPath(r.URL.Path, claims)
	if err != nil {
		if webhelper.ReturnError(w, r, err, httpStatus) {
			return
		}
	}

	var trackData TrackData
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&trackData)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	storage.DeepCopy(trackData, track)
	err = track.Update()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	json.NewEncoder(w).Encode(track)
	return
}

func DeleteTrack(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}
	track, err, httpStatus := getTrackByUrlPath(r.URL.Path, claims)
	if err != nil {
		if webhelper.ReturnError(w, r, err, httpStatus) {
			return
		}
	}
	err = track.Delete()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	var responseDetails webhelper.Response
	responseDetails.Message = "Record Successfully Deleted"
	json.NewEncoder(w).Encode(responseDetails)
	return
}
