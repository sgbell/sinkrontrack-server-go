package userLogin

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/objectbox/objectbox-go/objectbox"
	"golang.org/x/crypto/bcrypt"
)

/*
 * Need to re-write my connection to objectbox to make it an interface, so I
 * can get the unit tests up and going, independent of objectbox-go.
 * It should also allow me to replace objectbox-go later if I really want to.
 */

var executeCreateUser func(m *User) (*uint64, error)
var executeFindUser func(conditions []objectbox.Condition) ([]*User, error)
var executeSelectUser func(m *User) error
var executeDeleteUser func(m *User) error
var executeUpdateUser func(m *User) error

func (m *User) Insert() (*uint64, error) {
	return executeCreateUser(m)
}

func (m *User) Find(conditions ...objectbox.Condition) ([]*User, error) {
	return executeFindUser(conditions)
}

func (m *User) Select() error {
	return executeSelectUser(m)
}

func (m *User) Delete() error {
	return executeDeleteUser(m)
}

func (m *User) Update() error {
	return executeUpdateUser(m)
}

func TestCreateUser(t *testing.T) {
	t.Run("create new user returning id", func(t *testing.T) {
		user := User{}
		user.FirstName = "Test"
		user.LastName = "User"
		user.EmailAddress = "test@test.com"
		user.Password = "blah12345"
		user.Enabled = true
		user.AdminUser = false

		executeCreateUser = func(m *User) (*uint64, error) {
			return &[]uint64{1}[0], nil
		}
		id, _ := CreateUser(&user)
		if id == nil {
			t.Error("Expected user failed to created")
		}
	})

	t.Run("fail to create user due to blank password", func(t *testing.T) {
		user := User{}
		user.FirstName = "Test"
		user.LastName = "User"
		user.EmailAddress = "test@test.com"
		user.Enabled = true
		user.AdminUser = false

		executeCreateUser = func(m *User) (*uint64, error) {
			return &[]uint64{1}[0], nil
		}
		_, err := CreateUser(&user)
		if err == nil {
			t.Error("Expected User create failed due to no password")
		}
	})
	t.Run("Confirm when password fails encryption it throws error", func(t *testing.T) {
		user := User{}
		user.FirstName = "Test"
		user.LastName = "User"
		user.EmailAddress = "test@test.com"
		user.Enabled = true
		user.AdminUser = false
		user.Password = "blah12345"

		bcryptGenerateFromPassword = func(password []byte, cost int) ([]byte, error) {
			return nil, bcrypt.ErrMismatchedHashAndPassword
		}

		_, err := CreateUser(&user)
		if err == nil {
			t.Error("Expected User create failed due to password encryption error")
		}
	})
}

// Test the private functions first
func TestCheckPassword(t *testing.T) {
	t.Run("password and confirmPassword are blank", func(t *testing.T) {
		status, err := checkPassword("", "", true)
		if status == nil {
			t.Errorf("Want status '%d', got nil", status)
		}
		if err == nil {
			t.Errorf("Want error 'Blank Password', got nil")
		}
		if err.Error() != "Blank Password" {
			t.Errorf("Want error message 'Blank Password', got '%s'", err.Error())
		}
	})
	t.Run("password and confirmPassword are blank and checkBothBlank is false", func(t *testing.T) {
		status, err := checkPassword("", "", false)
		if status != nil || err != nil {
			t.Error("I'm seriously unsure why I would allow blank values")
		}
	})
	t.Run("password and confirmPassword are different, checkBothBlank is true", func(t *testing.T) {
		status, err := checkPassword("Random Password Value", "", false)
		if status != nil && *status != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, status)
		}
		if err != nil && err.Error() != "Passwords do not match" {
			t.Errorf("Want error message 'Passwords do not match', got '%s'", err.Error())
		}
		if err == nil {
			t.Errorf("Want error message 'Passwords do not match', got no error")
		}
	})
}

func TestCheckEmail(t *testing.T) {
	t.Run("emailAddress empty, checkEmailEmpty is true", func(t *testing.T) {
		status, err := checkEmail("", true)
		if status != nil && *status != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, status)
		}
		if status == nil {
			t.Errorf("Want status '%d', got nil", http.StatusBadRequest)
		}
		if err != nil && err.Error() != "Missing Email Address" {
			t.Errorf("Want error '%s', got '%s'", "Missing Email Address", err.Error())
		}
	})
	t.Run("emailAddress empty, checkEmailEmpty is false", func(t *testing.T) {
		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			var userlist []*User
			return userlist, nil
		}

		status, err := checkEmail("", false)
		if status != nil {
			t.Errorf("Want status nil, got '%d'", status)
		}
		if err != nil {
			t.Errorf("Want error nil, go '%s'", err.Error())
		}
	})
	t.Run("emailAddress does not exist, checkEmailEmpty is true", func(t *testing.T) {
		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			var userlist []*User
			return userlist, nil
		}

		status, err := checkEmail("test@test.com.au", true)
		if status != nil {
			t.Errorf("Want status nil, got '%d'", status)
		}
		if err != nil {
			t.Errorf("Want error nil, go '%s'", err.Error())
		}
	})
	t.Run("emailAddress exists", func(t *testing.T) {
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

		status, err := checkEmail("test@test.com.au", true)
		if *status != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, *status)
		}
		if err == nil {
			t.Errorf("Want error message '%s', got nil", "User already exists")
		} else if err.Error() != "User already exists" {
			t.Errorf("Want error message '%s', got '%s'", "User already exists", err.Error())
		}
	})
}

func TestCreateUserLogin(t *testing.T) {
	t.Run("no data passed to the function", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)
		responseRecorder := httptest.NewRecorder()
		CreateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm Bad Json throws an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com"`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		CreateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("no password passed to the function", func(t *testing.T) {
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com"}`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		CreateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})

	t.Run("passwords do not match passed to the function", func(t *testing.T) {
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com", "password":"testPassword1", "confirmPassword":"testPassword2"}`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		CreateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})

	t.Run("blank email passed to the function", func(t *testing.T) {
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"", "password":"testPassword1", "confirmPassword":"testPassword2"}`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		CreateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})

	t.Run("check email does not exist in database for another user", func(t *testing.T) {
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com.au", "password":"testPassword1", "confirmPassword":"testPassword1"}`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.Id = 2
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com.au"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		executeCreateUser = func(m *User) (*uint64, error) {
			return &[]uint64{1}[0], nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		CreateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
}

func TestCheckToken(t *testing.T) {
	t.Run("no token passed for checking", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)

		//request.AddCookie(&http.Cookie{Name:"token", Value:""})
		_, status := CheckToken(request)
		if status != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, status)
		}
	})
	t.Run("empty token passed for checking", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)

		request.AddCookie(&http.Cookie{Name: "token", Value: ""})

		/*	jwtParseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc) (*jwt.Token, error) {
			return &jwt.Token{Raw: "blah", Method: jwt.SigningMethodHS256, Claims: claims, Signature: "blah blah", Valid: false}, nil
		}*/

		_, status := CheckToken(request)
		if status != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, status)
		}
	})
	t.Run("invalid token passed for checking", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)

		request.AddCookie(&http.Cookie{Name: "token", Value: "blahblahblah"})

		jwtParseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc) (*jwt.Token, error) {
			return nil, jwt.ErrSignatureInvalid
		}

		_, status := CheckToken(request)
		if status != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, status)
		}
	})
	t.Run("invalid token passed for checking", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)

		request.AddCookie(&http.Cookie{Name: "token", Value: "blahblahblah"})

		jwtParseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc) (*jwt.Token, error) {
			return nil, jwt.ErrHashUnavailable
		}

		_, status := CheckToken(request)
		if status != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, status)
		}
	})
	t.Run("passed token is not valid", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)

		request.AddCookie(&http.Cookie{Name: "token", Value: ""})

		jwtParseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc) (*jwt.Token, error) {
			return &jwt.Token{Raw: "blah", Method: jwt.SigningMethodHS256, Claims: claims, Signature: "blah blah", Valid: false}, nil
		}

		_, status := CheckToken(request)
		if status != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, status)
		}
	})
	t.Run("passed token is valid", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/user", nil)

		request.AddCookie(&http.Cookie{Name: "token", Value: ""})

		jwtParseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc) (*jwt.Token, error) {
			return &jwt.Token{Raw: "blah", Method: jwt.SigningMethodHS256, Claims: claims, Signature: "blah blah", Valid: true}, nil
		}

		_, status := CheckToken(request)
		if status != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, status)
		}
	})
}

func TestDeleteUserLogin(t *testing.T) {
	t.Run("Delete User with no token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusUnauthorized
		}
		request := httptest.NewRequest("POST", "/user", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Confirm no user account specified on url for GET delete", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusOK
		}

		request := httptest.NewRequest("GET", "/user/delete/", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm no user account specified on url for DELETE ", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusOK
		}

		request := httptest.NewRequest("GET", "/user/", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Reject user id is of type string", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusOK
		}

		request := httptest.NewRequest("GET", "/user/boo1", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm user id 1 (Primary Admin account) is rejected from deletion", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusOK
		}

		request := httptest.NewRequest("GET", "/user/1", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Reject call when user does not exist", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusOK
		}

		request := httptest.NewRequest("GET", "/user/1", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}

	})
	t.Run("Confirm non admin user cannot delete another user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.Id = 2
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		request := httptest.NewRequest("DELETE", "/user/3", nil)
		responseRecorder := httptest.NewRecorder()

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusForbidden {
			t.Errorf("Want status '%d', got '%d'", http.StatusForbidden, responseRecorder.Code)
		}
	})
	t.Run("Confirm user is deleted by admin user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.Id = 2
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = true

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		request := httptest.NewRequest("DELETE", "/user/3", nil)
		responseRecorder := httptest.NewRecorder()

		executeDeleteUser = func(m *User) error {
			return nil
		}

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
	t.Run("Confirm user can delete themselves", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.Id = 2
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
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		request := httptest.NewRequest("DELETE", "/user/2", nil)
		responseRecorder := httptest.NewRecorder()

		executeDeleteUser = func(m *User) error {
			return nil
		}

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
	t.Run("Confirm error is thrown when user delete fails", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.Id = 2
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
			m.FirstName = ""
			m.LastName = ""
			m.EmailAddress = ""
			m.Enabled = false
			m.AdminUser = false

			return nil
		}

		request := httptest.NewRequest("DELETE", "/user/2", nil)
		responseRecorder := httptest.NewRecorder()

		executeDeleteUser = func(m *User) error {
			return errors.New("Storage error, can not delete")
		}

		DeleteUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
}

func TestListUsers(t *testing.T) {
	t.Run("List User with no token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusUnauthorized
		}
		request := httptest.NewRequest("GET", "/user", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})

	t.Run("When no user id is in the url, check if user is an Admin account", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return false
		}
		request := httptest.NewRequest("GET", "/user/", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusNotFound {
			t.Errorf("Want status '%d', got '%d'", http.StatusNotFound, responseRecorder.Code)
		}
	})
	t.Run("Confirm admin user can list all users", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return true
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.Id = 2
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			userlist = append(userlist, &user)
			return userlist, nil
		}

		request := httptest.NewRequest("GET", "/user/", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}

	})
	t.Run("Confirm user id is of type unsigned int", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return false
		}

		request := httptest.NewRequest("GET", "/user/FLAG", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm user thrown when does not exist", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return true
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = ""
			m.LastName = ""
			m.EmailAddress = ""
			m.Enabled = false
			m.AdminUser = false

			return errors.New("User Does not exist")
		}

		request := httptest.NewRequest("GET", "/user/2", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusNotFound {
			t.Errorf("Want status '%d', got '%d'", http.StatusNotFound, responseRecorder.Code)
		}
	})
	t.Run("Confirm non admin user cannot view another user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return false
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

		request := httptest.NewRequest("GET", "/user/2", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusForbidden {
			t.Errorf("Want status '%d', got '%d'", http.StatusForbidden, responseRecorder.Code)
		}
	})
	t.Run("Confirm admin user can view any user", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return true
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

		request := httptest.NewRequest("GET", "/user/2", nil)
		responseRecorder := httptest.NewRecorder()

		ListUsers(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestUpdateUserLogin(t *testing.T) {
	t.Run("Update User with no token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusUnauthorized
		}
		request := httptest.NewRequest("PATCH", "/user", nil)
		responseRecorder := httptest.NewRecorder()

		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Confirm Bad Json throws an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com"`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm user id missing in url, throws an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com"`
		request := httptest.NewRequest("POST", "/user", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm user id matches userlogin when user is not admin", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}

		storageIsAdminUser = func(username string) bool {
			return false
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 1
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com.au"}`
		request := httptest.NewRequest("POST", "/user/1", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Updating user's email address to an email already in the system", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		storageIsAdminUser = func(username string) bool {
			return false
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 1
			m.Enabled = true
			m.AdminUser = false

			return nil
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

		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com"}`
		request := httptest.NewRequest("POST", "/user/2", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm when password fails encryption it throws error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		storageIsAdminUser = func(username string) bool {
			return false
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 1
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			//userlist = append(userlist, &user)
			return userlist, nil
		}

		bcryptGenerateFromPassword = func(password []byte, cost int) ([]byte, error) {
			return nil, bcrypt.ErrMismatchedHashAndPassword
		}

		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com","password":"blahblahblah","confirmPassword":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/2", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm update failure throws an error", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		storageIsAdminUser = func(username string) bool {
			return false
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 1
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			//userlist = append(userlist, &user)
			return userlist, nil
		}

		bcryptGenerateFromPassword = func(password []byte, cost int) ([]byte, error) {
			return bcrypt.GenerateFromPassword(password, cost)
		}

		executeUpdateUser = func(m *User) error {
			return errors.New("Test update fails safely")
		}

		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com","password":"blahblahblah","confirmPassword":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/2", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("Confirm update success returns data", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com.au",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		storageIsAdminUser = func(username string) bool {
			return false
		}

		executeSelectUser = func(m *User) error {
			m.FirstName = "Test"
			m.LastName = "User"
			m.EmailAddress = "test@test.com.au"
			m.Id = 1
			m.Enabled = true
			m.AdminUser = false

			return nil
		}

		executeFindUser = func(conditions []objectbox.Condition) ([]*User, error) {
			user := User{}
			user.FirstName = "Test"
			user.LastName = "User"
			user.EmailAddress = "test@test.com"
			user.Enabled = true
			user.AdminUser = false

			var userlist []*User
			//userlist = append(userlist, &user)
			return userlist, nil
		}

		bcryptGenerateFromPassword = func(password []byte, cost int) ([]byte, error) {
			return bcrypt.GenerateFromPassword(password, cost)
		}

		executeUpdateUser = func(m *User) error {
			return nil
		}

		var data = `{"firstName":"Test","lastName":"User","emailAddress":"test@test.com","password":"blahblahblah","confirmPassword":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/2", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()
		UpdateUserLogin(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestSignIn(t *testing.T) {
	t.Run("Confirm Bad Json throws an error", func(t *testing.T) {
		os.Setenv("JWT_KEY", "")
		bcryptCompareHashAndPassword = func(hashedPassword []byte, password []byte) error {
			return errors.New("Password does not match stored hash")
		}
		var data = `{"username":"test@test.com","password":"blahblahblah"`
		request := httptest.NewRequest("POST", "/user/signin", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		Signin(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
	})
	t.Run("User matching email does not exist", func(t *testing.T) {
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
		os.Setenv("JWT_KEY", "")
		bcryptCompareHashAndPassword = func(hashedPassword []byte, password []byte) error {
			return errors.New("Password does not match stored hash")
		}

		var data = `{"username":"test@test.com","password":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/signin", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		Signin(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Password does not match stored user password", func(t *testing.T) {
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
		os.Setenv("JWT_KEY", "")
		bcryptCompareHashAndPassword = func(hashedPassword []byte, password []byte) error {
			return errors.New("Password does not match stored hash")
		}

		var data = `{"username":"test@test.com","password":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/signin", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		Signin(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("JWT_KEY is not defined in the environment", func(t *testing.T) {
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
		os.Setenv("JWT_KEY", "")
		bcryptCompareHashAndPassword = func(hashedPassword []byte, password []byte) error {
			return nil
		}

		var data = `{"username":"test@test.com","password":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/signin", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		Signin(responseRecorder, request)
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("Want status '%d', got '%d'", http.StatusInternalServerError, responseRecorder.Code)
		}
	})
	t.Run("Token Signing Succeeds", func(t *testing.T) {
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
		os.Setenv("JWT_KEY", "D5H5H65H56H5G4F3F3F32G")
		bcryptCompareHashAndPassword = func(hashedPassword []byte, password []byte) error {
			return nil
		}

		var data = `{"username":"test@test.com","password":"blahblahblah"}`
		request := httptest.NewRequest("POST", "/user/signin", strings.NewReader(data))
		responseRecorder := httptest.NewRecorder()

		Signin(responseRecorder, request)
		if responseRecorder.Code != http.StatusAccepted {
			t.Errorf("Want status '%d', got '%d'", http.StatusAccepted, responseRecorder.Code)
		}
	})
}

func TestRefreshToken(t *testing.T) {
	t.Run("RefreshToken call with no token", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			return nil, http.StatusUnauthorized
		}
		os.Setenv("JWT_KEY", "")
		request := httptest.NewRequest("PATCH", "/user", nil)
		responseRecorder := httptest.NewRecorder()

		RefreshToken(responseRecorder, request)
		if responseRecorder.Code != http.StatusUnauthorized {
			t.Errorf("Want status '%d', got '%d'", http.StatusUnauthorized, responseRecorder.Code)
		}
	})
	t.Run("Signing token succeeds", func(t *testing.T) {
		checkTokenVar = func(r *http.Request) (*Claims, int) {
			claims := &Claims{
				Username:       "test@test.com",
				StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
			}
			return claims, http.StatusOK
		}
		os.Setenv("JWT_KEY", "D5H5H65H56H5G4F3F3F32G")

		request := httptest.NewRequest("PATCH", "/user", nil)
		responseRecorder := httptest.NewRecorder()

		RefreshToken(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Errorf("Want status '%d', got '%d'", http.StatusOK, responseRecorder.Code)
		}
	})
}

func TestIsAdmin(t *testing.T) {
	t.Run("Given user is admin user, function should return true", func(t *testing.T) {
		claims := &Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		storageIsAdminUser = func(username string) bool {
			return true
		}

		if !isAdmin(claims) {
			t.Error("User should be declared admin user")
		}
	})
	t.Run("Given user is not an admin user, function should return false", func(t *testing.T) {
		claims := &Claims{
			Username:       "test@test.com",
			StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(60 * time.Minute).Unix()},
		}

		storageIsAdminUser = func(username string) bool {
			return false
		}

		if isAdmin(claims) {
			t.Error("User should be declared nonadmin user")
		}
	})
}

func TestGetJwtKey(t *testing.T) {
	t.Run("Given JWT_KEY length is 0, it should fail", func(t *testing.T) {
		os.Setenv("JWT_KEY", "")

		_, err := getJwtKey()
		if err == nil {
			t.Error("Should throw an error")
		}
	})
	t.Run("Given JWT_KEY length > 0, it should return key", func(t *testing.T) {
		os.Setenv("JWT_KEY", "D5H5H65H56H5G4F3F3F32G")

		jwtkey, _ := getJwtKey()
		if bytes.Compare(jwtkey, []byte("D5H5H65H56H5G4F3F3F32G")) != 0 {
			t.Error("Invalid JWT Key returned")
		}
	})
}
