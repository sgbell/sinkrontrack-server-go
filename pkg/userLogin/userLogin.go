package userLogin

import (
	"encoding/json"
	"errors"
	"mimpidev/sinkrontrack-server/internal/storage"
	"mimpidev/sinkrontrack-server/internal/webhelper"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserData struct {
	Id              uint64 `json:"id,omitempty"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	EmailAddress    string `json:"emailAddress"`
	Password        string `json:"password,omitempty"`
	PasswordConfirm string `json:"confirmPassword,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
	AdminUser       *bool  `json:"adminUser,omitempty"`
}

type UpdateUserData struct {
	FirstName       string `json:"firstName,omitempty"`
	LastName        string `json:"lastName,omitempty"`
	EmailAddress    string `json:"emailAddress,omitempty"`
	Password        string `json:"password,omitempty"`
	PasswordConfirm string `json:"confirmPassword,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
	AdminUser       *bool  `json:"adminUser,omitempty"`
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type User struct {
	storage.User
}

var jwtParseWithClaims = jwt.ParseWithClaims
var checkTokenVar = CheckToken
var storageIsAdminUser = storage.IsAdminUser
var bcryptGenerateFromPassword = bcrypt.GenerateFromPassword
var bcryptCompareHashAndPassword = bcrypt.CompareHashAndPassword

func CreateUser(m *User) (*uint64, error) {
	// Encrypt the password
	if m.Password == "" {
		return nil, errors.New("Blank Password")
	}
	passwordHash, err := bcryptGenerateFromPassword([]byte(m.Password), 14)
	if err != nil {
		return nil, err
	}

	m.Password = string(passwordHash)
	id, err := m.Insert()

	return id, err
}

// Going to use this function to confirm password
func checkPassword(password string, confirmPassword string, checkBothBlank bool) (*int, error) {
	if checkBothBlank &&
		password == "" &&
		confirmPassword == "" {
		var request = http.StatusBadRequest
		return &request, errors.New("Blank Password")
	}
	if password != confirmPassword {
		var request = http.StatusBadRequest
		return &request, errors.New("Passwords do not match")
	}

	return nil, nil
}

func checkEmail(emailAddress string, checkEmailEmpty bool) (*int, error) {
	// Search to confirm email address is not used by another user
	if checkEmailEmpty &&
		emailAddress == "" {
		var request = http.StatusBadRequest
		return &request, errors.New("Missing Email Address")
	}
	var user User
	findResults, _ := user.Find(storage.User_.EmailAddress.Equals(emailAddress, false))
	if len(findResults) > 0 {
		var request = http.StatusBadRequest
		return &request, errors.New("User already exists")
	}
	return nil, nil
}

func CreateUserLogin(w http.ResponseWriter, r *http.Request) {
	var userData UserData
	err := json.NewDecoder(r.Body).Decode(&userData)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if userData.Enabled == nil {
		userData.Enabled = &[]bool{true}[0]
	}

	userData.AdminUser = &[]bool{false}[0]

	httpCode, err := checkPassword(userData.Password, userData.PasswordConfirm, true)
	if webhelper.ReturnError(w, r, err, httpCode) {
		return
	}

	httpCode2, err := checkEmail(userData.EmailAddress, true)
	if webhelper.ReturnError(w, r, err, httpCode2) {
		return
	}

	user := new(User)

	user.FirstName = userData.FirstName
	user.LastName = userData.LastName
	user.EmailAddress = userData.EmailAddress
	user.Password = userData.Password
	user.Enabled = *userData.Enabled
	user.AdminUser = *userData.AdminUser

	id, err := CreateUser(user)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}

	var userOutput User
	userOutput.Id = *id
	err = userOutput.Select()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	userData.FirstName = userOutput.FirstName
	userData.LastName = userOutput.LastName
	userData.EmailAddress = userOutput.EmailAddress
	userData.Enabled = &userOutput.Enabled
	userData.AdminUser = &userOutput.AdminUser
	userData.Id = userOutput.Id
	userData.Password = ""
	userData.PasswordConfirm = ""
	json.NewEncoder(w).Encode(userData)
	return
}

func DeleteUserLogin(w http.ResponseWriter, r *http.Request) {
	claims, tokenResponse := checkTokenVar(r)
	if tokenResponse != http.StatusOK {
		w.WriteHeader(tokenResponse)
		return
	}

	regex := regexp.MustCompile("^/user/(?:delete/|)([^/]+)$")
	matches := regex.FindStringSubmatch(r.URL.Path)
	if len(matches) == 0 {
		err := errors.New("No user account specified")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
			return
		}
	}
	id, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		err := errors.New("User Account is invalid")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
			return
		}
	}
	if id == 1 {
		err := errors.New("Admin Account can not be deleted")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
			return
		}
	}

	// Check if user is not admin trying to delete another user
	var currentUser User
	users, _ := currentUser.Find(storage.User_.EmailAddress.Equals(claims.Username, false))
	if users != nil {
		for _, searchUser := range users {
			if !searchUser.AdminUser &&
				searchUser.Id != id {
				err := errors.New("Access Denied")
				if webhelper.ReturnError(w, r, err, &[]int{http.StatusForbidden}[0]) {
					return
				}
			}
		}
	}

	var user User
	user.Id = id
	err = user.Select()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusNotFound}[0]) {
		return
	}
	err = user.Delete()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	var response webhelper.Response
	response.Message = "Record Successfully Deleted"
	json.NewEncoder(w).Encode(response)
	return
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	regex := regexp.MustCompile("^/user/([^/]+)$")
	matches := regex.FindStringSubmatch(r.URL.Path)

	if len(matches) == 0 {
		if isAdmin(claims) {
			var user User
			users, err := user.Find(storage.User_.Id.GreaterOrEqual(1))
			if err != nil {
				err := errors.New("User Account is invalid")
				if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
					return
				}
			}
			var userList []*UserData
			for _, user := range users {
				var userData UserData
				userData.Id = user.Id
				userData.FirstName = user.FirstName
				userData.LastName = user.LastName
				userData.EmailAddress = user.EmailAddress
				userList = append(userList, &userData)
			}
			json.NewEncoder(w).Encode(userList)
			return
		} else {
			err := errors.New("User Account is invalid")
			if webhelper.ReturnError(w, r, err, &[]int{http.StatusNotFound}[0]) {
				return
			}
		}
	}
	id, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		err := errors.New("User Account is invalid")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
			return
		}
	}

	var user User
	user.Id = id
	err = user.Select()
	if err != nil {
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusNotFound}[0]) {
			return
		}
	}

	if !isAdmin(claims) &&
		claims.Username != user.EmailAddress {
		err := errors.New("Permission denied")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusForbidden}[0]) {
			return
		}
	}

	var userData UserData
	userData.Id = user.Id
	userData.FirstName = user.FirstName
	userData.LastName = user.LastName
	userData.EmailAddress = user.EmailAddress

	json.NewEncoder(w).Encode(userData)
	return
}

func UpdateUserLogin(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	var userData UpdateUserData
	err := json.NewDecoder(r.Body).Decode(&userData)
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	regex := regexp.MustCompile("^/user/([^/]+)$")
	matches := regex.FindStringSubmatch(r.URL.Path)
	if len(matches) == 0 {
		err := errors.New("No user account specified")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
			return
		}
	}
	id, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		err := errors.New("User Account is invalid")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
			return
		}
	}

	var user User
	user.Id = id
	err = user.Select()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	if user.EmailAddress != claims.Username &&
		!storageIsAdminUser(claims.Username) {
		err := errors.New("User can not modify another user account")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusUnauthorized}[0]) {
			return
		}
	}

	if userData.FirstName != "" {
		user.FirstName = userData.FirstName
	}
	if userData.LastName != "" {
		user.LastName = userData.LastName
	}
	if userData.EmailAddress != "" {
		user.EmailAddress = userData.EmailAddress
	}
	ec, err := checkEmail(userData.EmailAddress, false)
	if webhelper.ReturnError(w, r, err, ec) {
		return
	}

	// Need to check if email address already exists in the system and if so reject it
	sc, err := checkPassword(userData.Password, userData.PasswordConfirm, false)
	if webhelper.ReturnError(w, r, err, sc) {
		return
	}

	if userData.Password != "" {
		paswordHash, err := bcryptGenerateFromPassword([]byte(userData.Password), 14)
		if err != nil {
			err := errors.New("Failed to Encrypt Password")
			if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
				return
			}
		}
		user.Password = string(paswordHash)
	}
	if userData.Enabled != nil &&
		*userData.Enabled != user.Enabled {
		user.Enabled = *userData.Enabled
	}
	// Need to check if current user is an admin user, before allowing the AdminUser flag to be changed
	if !isAdmin(claims) {
		userData.AdminUser = &[]bool{false}[0]
	}

	err = user.Update()
	if webhelper.ReturnError(w, r, err, &[]int{http.StatusBadRequest}[0]) {
		return
	}
	var returnUserData UserData
	returnUserData.FirstName = user.FirstName
	returnUserData.LastName = user.LastName
	returnUserData.EmailAddress = user.EmailAddress
	returnUserData.Enabled = &user.Enabled
	returnUserData.Id = user.Id

	json.NewEncoder(w).Encode(returnUserData)
	return
}

func Signin(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	err := json.NewDecoder(r.Body).Decode(&creds)

	authType := r.Header.Get("X-Authentication-Type")

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var user User
	findResults, err := user.Find(storage.User_.EmailAddress.Equals(creds.Username, false))

	if err != nil {
		// Give them a different message than the actual issue
		err := errors.New("Failed to Encrypt Password")
		if webhelper.ReturnError(w, r, err, &[]int{http.StatusUnauthorized}[0]) {
			return
		}
	}

	for _, user := range findResults {
		err := bcryptCompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			expirationTime := time.Now().Add(60 * time.Minute)
			// Create the JWT claims, which includes the username and expiry time
			claims := &Claims{
				Username: creds.Username,
				StandardClaims: jwt.StandardClaims{
					// In JWT, the expiry time is expressed as unix milliseconds
					ExpiresAt: expirationTime.Unix(),
				},
			}
			if authType != "" {
				claims.StandardClaims = jwt.StandardClaims{
					Subject: authType,
					Id:      uuid.NewString(),
				}
			}
			// Declare the token with the algorithm used for signing, and the claims
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			jwtKey, err := getJwtKey()
			if err != nil {
				webhelper.ReturnError(w, r, err, &[]int{http.StatusInternalServerError}[0])
				return
			}
			tokenString, err := token.SignedString(jwtKey)
			if err != nil {
				// If there is an error in creating the JWT return an internal server error
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Finally, we set the client cookie for "token" as the JWT we just generated
			// we also set an expiry time which is the same as the token itself
			http.SetCookie(w, &http.Cookie{
				Name:    "token",
				Value:   tokenString,
				Expires: expirationTime,
			})
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
	w.WriteHeader(http.StatusUnauthorized)
	return
}

func RefreshToken(w http.ResponseWriter, r *http.Request) {
	claims, response := checkTokenVar(r)
	if response != 200 {
		w.WriteHeader(response)
		return
	}

	// Now, create a new token for the current use, with a renewed expiration time
	expirationTime := time.Now().Add(5 * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey, err := getJwtKey()
	if err != nil {
		webhelper.ReturnError(w, r, err, &[]int{http.StatusInternalServerError}[0])
		return
	}
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set the new token as the users `token` cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})
}

func getJwtKey() ([]byte, error) {
	jwtKey := []byte(os.Getenv("JWT_KEY"))
	if len(jwtKey) == 0 {
		return nil, errors.New("No JWT Key defined in environment")
	}
	return jwtKey, nil
}

func CheckToken(r *http.Request) (*Claims, int) {
	c, err := r.Cookie("token")
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, http.StatusUnauthorized
		}
		return nil, http.StatusBadRequest
	}
	tknStr := c.Value
	claims := &Claims{}
	tkn, err := jwtParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		jwtKey, err := getJwtKey()
		return jwtKey, err
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, http.StatusUnauthorized
		}
		return nil, http.StatusBadRequest
	}
	if !tkn.Valid {
		return nil, http.StatusUnauthorized
	}

	return claims, http.StatusOK
}

func isAdmin(claims *Claims) bool {
	// Need to hit up storage class to see if they are an admin user
	return storageIsAdminUser(claims.Username)
}
