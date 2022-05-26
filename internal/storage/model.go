package storage

//go:generate go run github.com/objectbox/objectbox-go/cmd/objectbox-gogen
type Track struct {
	Id               uint64
	Uuid             string `objectbox:"index:hash64"`
	Path             string `objectbox:"index:hash64"`
	ArtistName       string `objectbox:"index:hash64"`
	SongName         string `objectbox:"index:hash64"`
	AlbumName        string `objectbox:"index:hash64"`
	AlbumTrackNumber int
	TrackLength      int
}

type Playlist struct {
	Id                uint64
	Uuid              string `objectbox:"index:hash64"`
	Name              string
	CurrentTrackId    uint16
	Elapsed           int
	Tracks            []*Track
	ClientIdLock      string
	ClientLockExpires int64
	hashValue         string `objectbox:"-"`
}

type Friend struct {
	Id       uint64
	friendId string `objectbox:"index:hash64"`
}

type User struct {
	Id           uint64 // going to be an internal objectBoxId
	Uuid         string `objectbox:"index:hash64"`
	FirstName    string
	LastName     string
	EmailAddress string `objectbox:"index:hash64"`
	Password     string
	Enabled      bool
	AdminUser    bool
	LastPlaylist uint64
	Tracks       []*Track
	Playlists    []*Playlist
	Friends      []*Friend
}
