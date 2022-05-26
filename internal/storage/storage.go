package storage

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/google/uuid"
	"github.com/objectbox/objectbox-go/objectbox"
)

var Ob *objectbox.ObjectBox

type DataStorage interface {
	Insert() (*uint64, error)
	Update() (*uint64, error)
	Delete() error
	Select() error
	Find(conditions ...objectbox.Condition) (bool, error)
	AddPlaylist(p *Playlist) (*uint64, error)
	AddTrack(t *Track)
	Copy(src interface{})
}

func Initialize() *objectbox.ObjectBox {
	Ob, _ = objectbox.NewBuilder().Model(ObjectBoxModel()).Build()
	return Ob
}

func (m *User) Insert() (*uint64, error) {
	box := BoxForUser(Ob)
	findQuery := box.Query(User_.EmailAddress.Equals(m.EmailAddress, false))
	findResults, _ := findQuery.Count()
	if findResults > 0 {
		return nil, errors.New("User already exists")
	}
	m.Uuid = uuid.NewString()
	id, err := box.Put(m)

	return &id, err
}

func (m *User) Update() error {
	box := BoxForUser(Ob)
	_, err := box.Put(m)
	if err != nil {
		return err
	}
	errs := m.Select()

	return errs
}

func (m *User) Delete() error {
	// Moving all of the Code here.. so it can be datastorage agnostic
	box := BoxForUser(Ob)
	err := box.Remove(m)
	return err
}

func (m *User) Select() error {
	box := BoxForUser(Ob)

	if m.Id != 0 {
		loadUser, err := box.Get(m.Id)
		if err != nil {
			return err
		}
		m = loadUser
	} else if m.Uuid != "" {
		userList, _ := m.Find(User_.Uuid.Equals(m.Uuid, true))
		if len(userList) == 0 {
			return errors.New("Failed to Find User account")
		}
		loadUser, err := box.Get(userList[0].Id)
		if err != nil {
			return err
		}
		m = loadUser
	} else if m.EmailAddress != "" {
		userList, _ := m.Find(User_.EmailAddress.Equals(m.EmailAddress, true))
		if len(userList) == 0 {
			return errors.New("Failed to Find User account")
		}
		loadUser, err := box.Get(userList[0].Id)
		if err != nil {
			return err
		}
		m = loadUser
	}

	return nil
}

func (m *User) Exists() (bool, error) {
	box := BoxForUser(Ob)

	if m.Id != 0 {
		user, err := box.Get(m.Id)
		if user != nil {
			return true, nil
		}
		return false, err
	} else if m.Uuid != "" {
		userList, _ := m.Find(User_.Uuid.Equals(m.Uuid, true))
		if len(userList) == 0 {
			return false, errors.New("Failed to Find User account")
		}
	}
	return false, errors.New("User has no value to search for")
}

func IsAdminUser(username string) bool {
	box := BoxForUser(Ob)
	findQuery := box.Query(User_.EmailAddress.Equals(username, false),
		User_.AdminUser.Equals(true))
	findResults, _ := findQuery.Count()
	if findResults > 0 {
		return true
	}

	return false
}

// I'm not sure this is the best way to do a find on an object, but for testing my code, I think it's the best way to go.
func (m *User) Find(conditions ...objectbox.Condition) ([]*User, error) {
	box := BoxForUser(Ob)
	findQuery := box.Query(conditions...)
	users, err := findQuery.Find()
	return users, err
}

func UserAddPlaylist(m *User, p *Playlist) (*uint64, error) {
	box := BoxForUser(Ob)
	p.Uuid = uuid.NewString()
	m.Playlists = append(m.Playlists, p)
	index, err := box.Put(m)
	return &index, err
}

func (p *Playlist) Delete() error {
	box := BoxForPlaylist(Ob)
	err := box.Remove(p)
	return err
}

func (p *Playlist) Select() error {
	box := BoxForPlaylist(Ob)

	if p.Id != 0 {
		loadPlaylist, err := box.Get(p.Id)
		if err != nil {
			return err
		}
		p = loadPlaylist
	} else if p.Uuid != "" {
		playlistResult, _ := p.Find(Playlist_.Uuid.Equals(p.Uuid, true))
		if len(playlistResult) == 0 {
			return errors.New("Failed to Find Playlist")
		}
		loadPlaylist, err := box.Get(playlistResult[0].Id)
		if err != nil {
			return err
		}
		p = loadPlaylist
	}
	return errors.New("Missing Id")
}

func (p *Playlist) Exists() (bool, error) {
	box := BoxForPlaylist(Ob)

	if p.Id != 0 {
		loadPlaylist, err := box.Get(p.Id)
		if loadPlaylist != nil {
			return true, nil
		}
		return false, err
	} else if p.Uuid != "" {
		playlistResult, _ := p.Find(Playlist_.Uuid.Equals(p.Uuid, true))
		if len(playlistResult) == 0 {
			return false, errors.New("Failed to Find Playlist")
		}
		return true, nil
	}
	return false, errors.New("Missing id")
}

func (p *Playlist) Find(conditions ...objectbox.Condition) ([]*Playlist, error) {
	box := BoxForPlaylist(Ob)
	findQuery := box.Query(conditions...)
	playlists, err := findQuery.Find()
	return playlists, err
}

func (p *Playlist) Update() error {
	box := BoxForPlaylist(Ob)
	_, err := box.Put(p)
	return err
}

func PlaylistAddTrack(p *Playlist,t *Track) (*uint64, error) {
	box := BoxForPlaylist(Ob)
	t.Uuid = uuid.NewString()
	p.Tracks = append(p.Tracks, t)
	index, err := box.Put(p)
	return &index, err
}

func (t *Track) Find(conditions ...objectbox.Condition) ([]*Track, error) {
	box := BoxForTrack(Ob)
	findQuery := box.Query(conditions...)
	tracks, err := findQuery.Find()
	return tracks, err
}

func (t *Track) Delete() error {
	box := BoxForTrack(Ob)
	err := box.Remove(t)
	return err
}

func (t *Track) Update() error {
	box := BoxForTrack(Ob)
	_, err := box.Put(t)
	return err
}

func DeepCopy(src, dest interface{}) {
	// Copy all Fields
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	enc.Encode(src)
	dec.Decode(dest)
}
