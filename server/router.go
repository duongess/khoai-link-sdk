package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/duongess/khoai-link-protocol/core"
)

// Context chua toan bo metadata kem RequestID
type Context[T any] struct {
	Req       *http.Request
	RequestID string
	Body      T
}

func (c *Context[T]) Param(key string) string {
	return c.Req.PathValue(key)
}

func (c *Context[T]) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context[T]) QueryInt(key string, defaultVal int) int {
	valStr := c.Req.URL.Query().Get(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// HandlerFunc tra ve (data, httpStatus, error)
type HandlerFunc[T any] func(c *Context[T]) (any, int, error)

type EmptyBody struct{}

type Router struct {
	mux *http.ServeMux
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rt.mux.ServeHTTP(w, req)
}

func RegisterGET(rt *Router, path string, handler HandlerFunc[EmptyBody]) {
	register(rt, "GET", path, handler, false)
}

func RegisterPOST[T any](rt *Router, path string, handler HandlerFunc[T]) {
	register(rt, "POST", path, handler, true)
}

func RegisterPUT[T any](rt *Router, path string, handler HandlerFunc[T]) {
	register(rt, "PUT", path, handler, true)
}

func RegisterDELETE(rt *Router, path string, handler HandlerFunc[EmptyBody]) {
	register(rt, "DELETE", path, handler, false)
}

func register[T any](rt *Router, method, path string, handler HandlerFunc[T], hasBody bool) {
	rt.mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, req *http.Request) {
		reqID := getOrCreateRequestID(req)
		w.Header().Set("X-Request-ID", reqID)

		var body T
		if hasBody {
			req.Body = http.MaxBytesReader(w, req.Body, 1<<20)
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				msg := "Invalid JSON: " + err.Error()
				if err == io.EOF {
					msg = "Request body cannot be empty"
				}
				writeEnvelope(w, http.StatusBadRequest, core.NewErrorResponse(reqID, http.StatusBadRequest, msg))
				return
			}
		}

		ctx := &Context[T]{
			Req:       req,
			RequestID: reqID,
			Body:      body,
		}

		resData, statusCode, err := handler(ctx)
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		if err != nil {
			writeEnvelope(w, statusCode, core.NewErrorResponse(reqID, statusCode, err.Error()))
			return
		}

		writeEnvelope(w, statusCode, core.NewSuccessResponse(reqID, resData))
	})
}

func writeEnvelope(w http.ResponseWriter, httpStatus int, resp *core.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

func getOrCreateRequestID(r *http.Request) string {
	reqID := r.Header.Get("X-Request-ID")
	if reqID != "" {
		return reqID
	}
	bytes := make([]byte, 4)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("req_%d_%s", time.Now().Unix(), hex.EncodeToString(bytes))
}
