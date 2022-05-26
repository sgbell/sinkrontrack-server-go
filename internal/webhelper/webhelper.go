package webhelper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

type Response struct {
	Message string `json:"message"`
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	var response Response
	response.Message = "Welcome to sinkrontrack-server"
	json.NewEncoder(w).Encode(response)
	return
}

type Route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

func NewRoute(method, pattern string, handler http.HandlerFunc) {
	Routes = append(Routes, Route{method, regexp.MustCompile("^" + pattern + "$"), handler})
}

func Serve(w http.ResponseWriter, r *http.Request) {
	var allow []string

	// Force it to be a json only application
	body, _ := ioutil.ReadAll((r.Body))
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(body) > 0 && r.Header.Get("Content-type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, route := range Routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			if r.Method != route.method {
				allow = append(allow, route.method)
				continue
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, matches[1:])
			route.handler(w, r.WithContext(ctx))
			return
		}
	}
	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.NotFound(w, r)
}

type ctxKey struct{}

var Routes = []Route{}

func ReturnError(w http.ResponseWriter, r *http.Request, err error, httpCode *int) bool {
	if httpCode != nil &&
		err != nil {
		w.WriteHeader(*httpCode)
		var response Response
		response.Message = err.Error()
		json.NewEncoder(w).Encode(response)
		return true
	}
	return false
}

func GetUUidFromUrl(url string, reg_exp string) (*string, error) {

	regex := regexp.MustCompile(reg_exp)
	matches := regex.FindStringSubmatch(url)
	if len(matches) == 0 {
		return nil, errors.New("Url is invalid")
	}
	_, err := uuid.Parse(matches[1])

	if err == nil {
		return &matches[1], nil
	}

	return nil, errors.New("Url is invalid")
}
