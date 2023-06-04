package httpwriter

import (
	"encoding/json"
	"net/http"
)

// Use for swagger generator
type JSONResponseWithoutData struct {
	Message string `json:"message,omitempty" example:"Error or Success message if exist"`
}

// Use for swagger generator
type JSONResponseWithData struct {
	Message string        `json:"message,omitempty" example:"Error or Success message if exist"`
	Data    []interface{} `json:"data,omitempty" swaggertype:"object"`
}

type Writer interface {
	WriteError(w http.ResponseWriter, err error, status int)
	WriteSuccess(w http.ResponseWriter, message string, data interface{})
}

type JSONResponseWriter struct{}

type Response struct {
	Code    int               `json:"-"`
	Headers map[string]string `json:"-"`
	Message string            `json:"message,omitempty"`
	Data    interface{}       `json:"data,omitempty"`
}

func NewJSONResponseWriter() JSONResponseWriter {
	return JSONResponseWriter{}
}

func (w *JSONResponseWriter) WriteError(rw http.ResponseWriter, err error, status int) {
	response := Response{
		Code:    status,
		Message: err.Error(),
	}
	response.Code = status
	response.Message = err.Error()

	w.write(rw, response)
}

func (w *JSONResponseWriter) WriteSuccess(rw http.ResponseWriter, message string, data interface{}) {
	response := Response{}
	if message != "" {
		response.Message = message
	} else {
		response.Message = "success"
	}

	if data != nil {
		response.Data = data
	}

	response.Code = 200

	w.write(rw, response)
}

func (w *JSONResponseWriter) write(rw http.ResponseWriter, r Response) {
	body, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(r.Code)

	for k, v := range r.Headers {
		rw.Header().Add(k, v)
	}

	if _, err := rw.Write(body); err != nil {
		panic(err)
	}
}
