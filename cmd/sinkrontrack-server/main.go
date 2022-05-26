package main

import (
	"fmt"
	"mimpidev/sinkrontrack-server/internal/storage"
	"mimpidev/sinkrontrack-server/internal/webhelper"
	"mimpidev/sinkrontrack-server/pkg/playlist"
	"mimpidev/sinkrontrack-server/pkg/userLogin"
	"net/http"
	"os"
	"strconv"
)

func buildRoutes() {
	webhelper.NewRoute("GET", "/", webhelper.RootHandler)
	webhelper.NewRoute("POST", "/users(/|)", userLogin.CreateUserLogin)
	webhelper.NewRoute("DELETE", "/users/([^/]+)", userLogin.DeleteUserLogin)
	webhelper.NewRoute("PATCH", "/users/([^/]+)", userLogin.UpdateUserLogin)
	webhelper.NewRoute("GET", "/users/([^/]+)", userLogin.ListUsers)
	webhelper.NewRoute("GET", "/users(/|)", userLogin.ListUsers)
	webhelper.NewRoute("POST", "/users/signin", userLogin.Signin)
	webhelper.NewRoute("POST", "/users/refreshToken", userLogin.RefreshToken)
	webhelper.NewRoute("GET", "/playlists(/|)", playlist.ListPlaylist)
	webhelper.NewRoute("GET", "/playlists/([^/]+)", playlist.GetPlaylist)
	webhelper.NewRoute("POST", "/playlists(/|)", playlist.CreatePlaylist)
	webhelper.NewRoute("PATCH", "/playlists/([^/]+)", playlist.UpdatePlaylist)
	webhelper.NewRoute("DELETE", "/playlists/([^/]+)", playlist.DeletePlaylist)
	webhelper.NewRoute("POST", "/playlists/([^/]+)/track", playlist.AddTrack)
	webhelper.NewRoute("PATCH", "/tracks/([^/]+)", playlist.UpdateTrack)
	webhelper.NewRoute("DELETE", "/tracks/([^/]+)", playlist.DeleteTrack)

}

func initializeAdminUser() {
	user := userLogin.User{}

	user.Id = 1
	exists, err := user.Exists()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	if !exists {
		if os.Getenv("ADMIN_EMAIL") == "" || os.Getenv("ADMIN_PASSWORD") == "" {
			fmt.Println("No Admin User Defined")
			os.Exit(2)
		}
		user.FirstName = "Admin"
		user.LastName = "User"
		user.EmailAddress = os.Getenv("ADMIN_EMAIL")
		user.Password = os.Getenv("ADMIN_PASSWORD")
		user.Enabled = true
		user.AdminUser = true
		id, err := userLogin.CreateUser(&user)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(2)
		}
		fmt.Println("Created Admin User: " + strconv.FormatUint(*id, 10))
	}
}

func main() {
	storage.Initialize()
	defer storage.Ob.Close()

	jwtKey := []byte(os.Getenv("JWT_KEY"))
	if len(jwtKey) == 0 {
		fmt.Println("No JWT Key defined in environment")
		os.Exit(1)
	}

	initializeAdminUser()

	buildRoutes()
	http.HandleFunc("/", webhelper.Serve)
	http.ListenAndServe(":9999", nil)
}
