package rest_test

import (
	"encoding/json"
	"errors"
	"maps"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/andrey-kalistratov/threat-info-service/internal/config"
	"github.com/andrey-kalistratov/threat-info-service/internal/logging/loggingtest"
	"github.com/andrey-kalistratov/threat-info-service/internal/rest"
	"github.com/andrey-kalistratov/threat-info-service/internal/rest/generated"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
	threatrest "github.com/andrey-kalistratov/threat-info-service/internal/threat/rest"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat/rest/mocks"
)

const canonicalMD5 = "d41d8cd98f00b204e9800998ecf8427e"

func TestServer_InvalidRequests(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		route  string
		body   string
	}{
		{
			name:   "Malformed JSON",
			method: http.MethodPost,
			path:   "/info",
			route:  "/info",
			body:   `{"name":"private-body-marker",`,
		},
		{
			name:   "Invalid GET path MD5",
			method: http.MethodGet,
			path:   "/info/invalid",
			route:  "/info/{md5}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange.
			wantStatus := http.StatusBadRequest
			wantResponse := generated.ErrorResponse{
				Message: generated.NewOptString("Bad Request"),
			}
			wantMediaType := "application/json"
			wantLog := loggingtest.Entry{
				Message: "HTTP request",
				Attrs: map[string]any{
					"method": tt.method,
					"route":  tt.route,
					"status": int64(wantStatus),
				},
			}

			service := mocks.NewMockThreatService(t)

			logger := &loggingtest.Recorder{}
			handler := newHandler(t, service, logger, 65536)

			req := newRequest(t, tt.method, tt.path, tt.body)
			recorder := httptest.NewRecorder()

			// Act.
			handler.ServeHTTP(recorder, req)

			// Assert.
			require.Equal(t, wantStatus, recorder.Code)
			requireErrorResponse(t, recorder, wantResponse, wantMediaType)
			requireRequestLog(t, logger, wantLog)
			require.Empty(t, logger.Errors)
		})
	}
}

func TestServer_BodyLimit(t *testing.T) {
	// Arrange.
	payload := &generated.ThreatData{
		Name: strings.Repeat("x", 256),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	wantStatus := http.StatusRequestEntityTooLarge

	service := mocks.NewMockThreatService(t)

	logger := &loggingtest.Recorder{}
	handler := newHandler(t, service, logger, 64)

	req := newRequest(t, http.MethodPut, "/info/"+canonicalMD5, string(body))
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	recorder := httptest.NewRecorder()

	// Act.
	handler.ServeHTTP(recorder, req)

	// Assert.
	require.Equal(t, wantStatus, recorder.Code)
	require.Empty(t, recorder.Body.String())
	require.Empty(t, logger.Errors)
}

func TestServer_ResponseWriteError(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		status  int
		request *generated.Threat
	}{
		{
			name: "Successful operation response",
			request: &generated.Threat{
				MD5:  generated.MD5(canonicalMD5),
				Name: "body-marker",
			},
			status: http.StatusCreated,
		},
		{
			name:   "Transport error response",
			body:   `{`,
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange.
			wantStatuses := []int{tt.status}
			wantLog := loggingtest.Entry{
				Message: "HTTP request",
				Attrs: map[string]any{
					"method": http.MethodPost,
					"route":  "/info",
					"status": int64(tt.status),
				},
			}

			writer := &failingWriter{
				header: make(http.Header),
				err:    errors.New("write failure"),
			}

			service := mocks.NewMockThreatService(t)
			body := tt.body
			if tt.request != nil {
				value := &threat.Threat{
					MD5:          canonicalMD5,
					Name:         "body-marker",
					MalwareTypes: []threat.MalwareType{},
				}
				service.EXPECT().Create(mock.Anything, value).Return(nil).Once()
				encoded, err := json.Marshal(tt.request)
				require.NoError(t, err)
				body = string(encoded)
			}

			logger := &loggingtest.Recorder{}
			handler := newHandler(t, service, logger, 65536)

			req := newRequest(t, http.MethodPost, "/info", body)
			req.URL.RawQuery = "secret=query-marker"
			req.Header.Set("X-Secret", "header-marker")

			// Act.
			handler.ServeHTTP(writer, req)

			// Assert.
			require.Equal(t, wantStatuses, writer.statuses)
			require.Equal(t, 1, writer.writes)
			require.Len(t, logger.Errors, 1)
			requireRequestLog(t, logger, wantLog)
		})
	}
}

type failingWriter struct {
	header   http.Header
	statuses []int
	writes   int
	err      error
}

func (w *failingWriter) Header() http.Header { return w.header }

func (w *failingWriter) WriteHeader(status int) {
	w.statuses = append(w.statuses, status)
}

func (w *failingWriter) Write(_ []byte) (int, error) {
	w.writes++
	return 0, w.err
}

func newHandler(t *testing.T, service *mocks.MockThreatService, logger *loggingtest.Recorder, maxBodyBytes int64) http.Handler {
	t.Helper()

	cfg := config.HTTP{
		Host:           "127.0.0.1",
		Port:           8080,
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		MaxHeaderBytes: 4096,
		MaxBodyBytes:   maxBodyBytes,
	}

	server, err := rest.NewServer(&cfg, logger, threatrest.NewHandler(logger, service))
	require.NoError(t, err)

	return server.Handler
}

func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

func requireRequestLog(t *testing.T, logger *loggingtest.Recorder, want loggingtest.Entry) {
	t.Helper()

	require.Len(t, logger.Info, 1)

	entry := logger.Info[0]
	duration, ok := entry.Attrs["duration"].(time.Duration)
	require.True(t, ok)
	require.GreaterOrEqual(t, duration, time.Duration(0))

	entry.Attrs = maps.Clone(entry.Attrs)
	delete(entry.Attrs, "duration")
	require.Equal(t, want, entry)
}

func requireErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantBody generated.ErrorResponse, wantMediaType string) {
	t.Helper()

	var actual generated.ErrorResponse

	mediaType, _, err := mime.ParseMediaType(recorder.Header().Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, wantMediaType, mediaType)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &actual))

	require.Equal(t, wantBody, actual)
}
