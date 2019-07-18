package httperr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"

	errors "golang.org/x/xerrors"
)

// StatusCoder returns an HTTP status code
type StatusCoder interface {
	StatusCode() int
}

type httpError struct {
	code int
	err  error
}

func (e *httpError) Error() string {
	status := http.StatusText(e.code)
	if e.err == nil {
		return fmt.Sprintf("%d %s", e.code, status)
	}
	return fmt.Sprintf("%d %s: %q", e.code, status, e.err)
}
func (e *httpError) StatusCode() int {
	return e.code
}
func (e *httpError) Unwrap() error {
	return e.err
}

// New creates a new HTTP error
func New(code int, err error) error {
	return &httpError{err: err, code: code}
}

// Errorf creates a new HTTP error by formating a message
func Errorf(code int, format string, args ...interface{}) error {
	return &httpError{
		err:  errors.Errorf(format, args...),
		code: code,
	}
}

// BadRequest creates an HTTP 500 error
func BadRequest(err error) error {
	return New(http.StatusBadRequest, err)
}

// InternalServerError creates an HTTP 500 error
func InternalServerError(err error) error {
	return New(http.StatusInternalServerError, err)
}

// NotFound creates an HTTP 404 error
func NotFound(err error) error {
	return New(http.StatusNotFound, err)
}

// MethodNotAllowed creates an HTTP 405 error
func MethodNotAllowed(err error) error {
	return New(http.StatusMethodNotAllowed, err)
}

// Response is a response message
type Response struct {
	Message    string `json:"message"`
	Error      string `json:"error"`
	StatusCode int    `json:"statusCode"`
}

func (e *httpError) MarshalJSON() ([]byte, error) {
	err := http.StatusText(e.code)
	msg := err
	if e.err != nil {
		msg = e.err.Error()
	}
	return json.Marshal(Response{
		Message:    msg,
		Error:      err,
		StatusCode: e.code,
	})
}

// IsInformational checks if code is HTTP informational code
func IsInformational(code int) bool {
	return http.StatusContinue <= code && code < http.StatusOK
}

// IsSuccess checks if code is HTTP success code
func IsSuccess(code int) bool {
	return http.StatusOK <= code && code < http.StatusMultipleChoices
}

// IsRedirect checks if code is HTTP redirect
func IsRedirect(code int) bool {
	return http.StatusMultipleChoices <= code && code < http.StatusBadRequest
}

// IsClientError checks if code is HTTP client error
func IsClientError(code int) bool {
	return http.StatusBadRequest <= code && code < http.StatusInternalServerError
}

// IsServerError checks if code is HTTP server error
func IsServerError(code int) bool {
	return http.StatusInternalServerError <= code && code < 600
}

// IsError checks if code is any HTTP error code
func IsError(code int) bool {
	return http.StatusBadRequest <= code && code < 600
}

// FromResponse creates a new HTTP error from a response
func FromResponse(r *http.Response) error {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return New(r.StatusCode, errors.Errorf("Failed to read response body: %q", err))
	}
	mediatype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch mediatype {
	case "text/plain", "text/html", "text/xml":
		return New(r.StatusCode, errors.New(string(data)))
	case "application/json":
		fallthrough
	default:
		var tmp Response
		if err := json.Unmarshal(data, &tmp); err != nil {
			return New(r.StatusCode, errors.Errorf("Error parsing response: %s", err))
		}
		return New(r.StatusCode, errors.New(tmp.Message))
	}
}

// RespondJSON sends a JSON encoded HTTP response
func RespondJSON(w http.ResponseWriter, x interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err, ok := x.(error); ok {
		code := http.StatusInternalServerError
		if coder, ok := err.(StatusCoder); ok {
			code = coder.StatusCode()
		}
		w.WriteHeader(code)
		if u, ok := err.(json.Unmarshaler); ok {
			return enc.Encode(u)
		}
		return enc.Encode(New(code, err))
	}
	w.WriteHeader(http.StatusOK)
	return enc.Encode(x)
}
