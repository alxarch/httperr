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

func BadRequest(err error) error {
	return New(http.StatusBadRequest, err)
}
func InternalServerError(err error) error {
	return New(http.StatusInternalServerError, err)
}
func NotFound(err error) error {
	return New(http.StatusNotFound, err)
}
func MethodNotAllowed(err error) error {
	return New(http.StatusMethodNotAllowed, err)
}

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

func IsInformational(code int) bool {
	return http.StatusContinue <= code && code < http.StatusOK
}
func IsSuccess(code int) bool {
	return http.StatusOK <= code && code < http.StatusMultipleChoices
}
func IsRedirect(code int) bool {
	return http.StatusMultipleChoices <= code && code < http.StatusBadRequest
}
func IsClientError(code int) bool {
	return http.StatusBadRequest <= code && code < http.StatusInternalServerError
}
func IsServerError(code int) bool {
	return http.StatusInternalServerError <= code && code < 600
}
func IsError(code int) bool {
	return http.StatusBadRequest <= code && code < 600
}

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
