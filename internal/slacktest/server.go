// Package slacktest is a lightweight fake Slack Web API server for tests.
// It's an httptest.Server that understands enough of Slack's Web API shape
// (POST /<method>, form-encoded params, {"ok":true/false,...} JSON, the
// three-step external upload flow) to drive real end-to-end tests of
// slack-cli — either in-process against the cobra command tree, or as a
// true black-box test of the compiled binary (see e2e/).
//
// Common methods (auth.test, chat.postMessage/update/delete,
// conversations.create/list/info, users.info/list/lookupByEmail, the
// files.getUploadURLExternal/completeUploadExternal pair plus the raw
// upload POST) have realistic default responses. Anything else defaults to
// a bare {"ok":true} unless a test overrides it — see Handle, QueueResponse,
// QueueError, and QueueRateLimited.
package slacktest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
)

// Handler produces one response for a single call to a Slack method.
type Handler func(r *http.Request) (status int, body any)

// Call records one request the server received, decoded and normalized to
// look the same whether the client sent form-encoding or a raw upload.
type Call struct {
	Method string
	Form   map[string]string // form-encoded params, single-valued
}

// Server is a fake Slack Web API server. Zero value is not usable; use New.
type Server struct {
	*httptest.Server

	// Token, if set, is the bearer token every /<method> request must
	// present; a mismatch responds with {"ok":false,"error":"invalid_auth"}.
	// Uploads to /_upload/... are unauthenticated, matching real Slack.
	Token string

	mu        sync.Mutex
	calls     []Call
	queues    map[string][]Handler // one-shot, consumed FIFO before permanent/default
	permanent map[string]Handler
	uploads   map[string][]byte // file id -> uploaded bytes, from the raw upload endpoint
	seq       int
}

// New starts a fake Slack API server. Call Close (via the embedded
// *httptest.Server) when done.
func New() *Server {
	s := &Server{
		queues:    map[string][]Handler{},
		permanent: map[string]Handler{},
		uploads:   map[string][]byte{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/_upload/", s.handleRawUpload)
	mux.HandleFunc("/", s.handleMethod)
	s.Server = httptest.NewServer(mux)
	return s
}

// Calls returns every request received so far, in order.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Call, len(s.calls))
	copy(out, s.calls)
	return out
}

// CallsFor returns every recorded call to the given method, in order.
func (s *Server) CallsFor(method string) []Call {
	var out []Call
	for _, c := range s.Calls() {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

// UploadedFile returns the bytes received for a file id via the raw upload
// endpoint (i.e. after a files.getUploadURLExternal + POST flow), and
// whether anything was uploaded under that id.
func (s *Server) UploadedFile(fileID string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.uploads[fileID]
	return b, ok
}

// Handle installs a permanent handler for method, replacing the built-in
// default. It's consulted after any one-shot Queue* responses for the same
// method are exhausted.
func (s *Server) Handle(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permanent[method] = h
}

// QueueResponse queues a single one-shot response for the next call to
// method (FIFO across multiple queued responses).
func (s *Server) QueueResponse(method string, status int, body any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[method] = append(s.queues[method], func(*http.Request) (int, any) { return status, body })
}

// QueueError queues a Slack-shaped {"ok":false,"error":code} response
// (HTTP 200, as Slack itself sends API errors) for the next call to method.
func (s *Server) QueueError(method, code string) {
	s.QueueResponse(method, http.StatusOK, map[string]any{"ok": false, "error": code})
}

// QueueRateLimited queues an HTTP 429 with a Retry-After header and Slack's
// real {"ok":false,"error":"ratelimited"} body for the next call to method,
// to exercise the client's retry behavior end-to-end.
func (s *Server) QueueRateLimited(method string, retryAfterSeconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[method] = append(s.queues[method], func(*http.Request) (int, any) {
		return http.StatusTooManyRequests, retryAfterMarker(retryAfterSeconds)
	})
}

// retryAfterMarker is a body type handleMethod recognizes to also set the
// Retry-After header; queued Handlers only get to return a body, not
// headers, and rate limiting is the one built-in case that needs one.
type retryAfterMarker int

func (s *Server) handleMethod(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimPrefix(r.URL.Path, "/")

	if s.Token != "" {
		if got := r.Header.Get("Authorization"); got != "Bearer "+s.Token {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "invalid_auth"})
			return
		}
	}

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_form"})
		return
	}
	form := map[string]string{}
	for k := range r.PostForm {
		form[k] = r.PostForm.Get(k)
	}

	s.mu.Lock()
	s.calls = append(s.calls, Call{Method: method, Form: form})
	var h Handler
	if q := s.queues[method]; len(q) > 0 {
		h, s.queues[method] = q[0], q[1:]
	} else if perm, ok := s.permanent[method]; ok {
		h = perm
	}
	s.mu.Unlock()

	if h == nil {
		writeJSON(w, http.StatusOK, s.defaultResponse(method, form))
		return
	}
	status, body := h(r)
	if ra, ok := body.(retryAfterMarker); ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(ra)))
		writeJSON(w, status, map[string]any{"ok": false, "error": "ratelimited"})
		return
	}
	writeJSON(w, status, body)
}

func (s *Server) handleRawUpload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/_upload/")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad multipart body", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.uploads[id] = data
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// nextID returns a monotonically increasing counter, used to build
// realistic-looking Slack ids and message timestamps. Callers must hold
// s.mu.
func (s *Server) nextIDLocked() int {
	s.seq++
	return s.seq
}

func (s *Server) nextTS() string {
	s.mu.Lock()
	n := s.nextIDLocked()
	s.mu.Unlock()
	return fmt.Sprintf("1700000000.%06d", n)
}

func (s *Server) nextID(prefix string) string {
	s.mu.Lock()
	n := s.nextIDLocked()
	s.mu.Unlock()
	return fmt.Sprintf("%s%07d", prefix, n)
}
