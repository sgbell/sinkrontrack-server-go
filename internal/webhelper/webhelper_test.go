package webhelper

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReturnError(t *testing.T) {
	t.Run("no data passed to the function", func(t *testing.T) {
		// Request values are not used in this function, so any data will be fine
		request := httptest.NewRequest("POST", "/", nil)
		responseRecorder := httptest.NewRecorder()
		var err error = nil
		var httpCode *int = nil
		if ReturnError(responseRecorder, request, err, httpCode) {
			t.Errorf("Expected returnError result to be false")
		}
	})

	t.Run("pass httpCode value but no error message", func(t *testing.T) {
		// Request values are not used in this function, so any data will be fine
		request := httptest.NewRequest("POST", "/", nil)
		responseRecorder := httptest.NewRecorder()
		var err error = nil
		var httpCode *int = &[]int{http.StatusBadRequest}[0]
		if ReturnError(responseRecorder, request, err, httpCode) {
			t.Errorf("Expected returnError result to be false")
		}
	})

	t.Run("pass error value but no http status code", func(t *testing.T) {
		// Request values are not used in this function, so any data will be fine
		request := httptest.NewRequest("POST", "/", nil)
		responseRecorder := httptest.NewRecorder()
		var err error = errors.New("Error thrown")
		var httpCode *int = nil
		if ReturnError(responseRecorder, request, err, httpCode) {
			t.Errorf("Expected returnError result to be false")
		}
	})

	t.Run("pass error value and http status code", func(t *testing.T) {
		// Request values are not used in this function, so any data will be fine
		request := httptest.NewRequest("POST", "/", nil)
		responseRecorder := httptest.NewRecorder()
		var err error = errors.New("Error thrown")
		var httpCode *int = &[]int{http.StatusBadRequest}[0]
		if !ReturnError(responseRecorder, request, err, httpCode) {
			t.Errorf("Expected Error to be thrown back in the response")
		}
		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("Want status '%d', got '%d'", http.StatusBadRequest, responseRecorder.Code)
		}
		var responseBody Response
		json.NewDecoder(responseRecorder.Body).Decode(&responseBody)
		if responseBody.Message != "Error thrown" {
			t.Errorf("Want error message '%s', got '%s'", "Error thrown", responseBody.Message)
		}
	})
}
