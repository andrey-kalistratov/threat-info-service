package rest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/andrey-kalistratov/threat-info-service/internal/rest/generated"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
	threatrest "github.com/andrey-kalistratov/threat-info-service/internal/threat/rest"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat/rest/mocks"
)

const (
	canonicalMD5 = "d41d8cd98f00b204e9800998ecf8427e"
	upperMD5     = "D41D8CD98F00B204E9800998ECF8427E"
)

func TestHandler_CreateThreat(t *testing.T) {
	tests := []struct {
		name string
		body generated.Threat
		want threat.Threat
	}{
		{
			name: "All fields",
			body: generated.Threat{
				MD5:          generated.MD5(upperMD5),
				Name:         "Trojan",
				Description:  generated.NewOptString("Collects data"),
				Confidence:   generated.NewOptInt32(0),
				MalwareTypes: []generated.MalwareType{generated.MalwareTypeTrojan, generated.MalwareTypeSpyware},
			},
			want: threat.Threat{
				MD5:          canonicalMD5,
				Name:         "Trojan",
				Description:  "Collects data",
				Confidence:   new(0),
				MalwareTypes: []threat.MalwareType{threat.MalwareTypeTrojan, threat.MalwareTypeSpyware},
			},
		},
		{
			name: "Absent optional fields",
			body: generated.Threat{
				MD5:  generated.MD5(upperMD5),
				Name: "Trojan",
			},
			want: threat.Threat{
				MD5:          canonicalMD5,
				Name:         "Trojan",
				MalwareTypes: []threat.MalwareType{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange.
			wantStatus := http.StatusCreated
			wantResponse := tt.body
			wantResponse.MD5 = canonicalMD5
			wantResponse.Description = generated.NewOptString(tt.want.Description)

			service := mocks.NewMockThreatService(t)
			service.EXPECT().Create(mock.Anything, &tt.want).Return(nil).Once()

			handler := newHandler(t, service)

			req := newJSONRequest(t, http.MethodPost, "/info", &tt.body)
			recorder := httptest.NewRecorder()

			// Act.
			handler.ServeHTTP(recorder, req)

			// Assert.
			require.Equal(t, wantStatus, recorder.Code)
			require.Equal(t, &wantResponse, readThreatResponse(t, recorder))
		})
	}
}

func TestHandler_UpdateThreat(t *testing.T) {
	// Arrange.
	body := &generated.ThreatData{
		Name: "Worm",
	}

	wantStatus := http.StatusOK
	wantResponse := &generated.Threat{
		MD5:         generated.MD5(canonicalMD5),
		Name:        "Worm",
		Description: generated.NewOptString(""),
	}
	want := &threat.Threat{
		MD5:          canonicalMD5,
		Name:         "Worm",
		MalwareTypes: []threat.MalwareType{},
	}

	service := mocks.NewMockThreatService(t)
	service.EXPECT().Update(mock.Anything, want).Return(nil).Once()

	handler := newHandler(t, service)

	req := newJSONRequest(t, http.MethodPut, "/info/"+upperMD5, body)
	recorder := httptest.NewRecorder()

	// Act.
	handler.ServeHTTP(recorder, req)

	// Assert.
	require.Equal(t, wantStatus, recorder.Code)
	require.Equal(t, wantResponse, readThreatResponse(t, recorder))
}

func TestHandler_ServiceErrors(t *testing.T) {
	internalError := errors.New("private database diagnostic")
	value := &threat.Threat{
		MD5:          canonicalMD5,
		Name:         "Trojan",
		MalwareTypes: []threat.MalwareType{},
	}
	tests := []struct {
		name    string
		method  string
		err     error
		status  int
		message string
		path    string
		body    any
	}{
		{
			name: "Create wrapped conflict",
			path: "/info",
			body: &generated.Threat{
				MD5:  generated.MD5(canonicalMD5),
				Name: "Trojan",
			},
			method:  http.MethodPost,
			err:     fmt.Errorf("insert: %w", threat.ErrAlreadyExists),
			status:  http.StatusConflict,
			message: "Threat already exists",
		},
		{
			name: "Create internal error",
			path: "/info",
			body: &generated.Threat{
				MD5:  generated.MD5(canonicalMD5),
				Name: "Trojan",
			},
			method:  http.MethodPost,
			err:     internalError,
			status:  http.StatusInternalServerError,
			message: "Internal Server Error",
		},
		{
			name:    "Get wrapped not found",
			path:    "/info/" + canonicalMD5,
			method:  http.MethodGet,
			err:     fmt.Errorf("select: %w", threat.ErrNotFound),
			status:  http.StatusNotFound,
			message: "Threat not found",
		},
		{
			name:    "Get internal error",
			path:    "/info/" + canonicalMD5,
			method:  http.MethodGet,
			err:     internalError,
			status:  http.StatusInternalServerError,
			message: "Internal Server Error",
		},
		{
			name: "Update wrapped not found",
			path: "/info/" + canonicalMD5,
			body: &generated.ThreatData{
				Name: "Trojan",
			},
			method:  http.MethodPut,
			err:     fmt.Errorf("update: %w", threat.ErrNotFound),
			status:  http.StatusNotFound,
			message: "Threat not found",
		},
		{
			name: "Update internal error",
			path: "/info/" + canonicalMD5,
			body: &generated.ThreatData{
				Name: "Trojan",
			},
			method:  http.MethodPut,
			err:     internalError,
			status:  http.StatusInternalServerError,
			message: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange.
			wantResponse := generated.ErrorResponse{
				Message: generated.NewOptString(tt.message),
			}

			service := mocks.NewMockThreatService(t)
			expectServiceError(service, tt.method, value, tt.err)

			handler := newHandler(t, service)

			req := newJSONRequest(t, tt.method, tt.path, tt.body)
			recorder := httptest.NewRecorder()

			// Act.
			handler.ServeHTTP(recorder, req)

			// Assert.
			require.Equal(t, tt.status, recorder.Code)
			var actual generated.ErrorResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &actual))
			require.Equal(t, wantResponse, actual)
		})
	}
}

func readThreatResponse(t *testing.T, recorder *httptest.ResponseRecorder) *generated.Threat {
	t.Helper()

	var actual generated.Threat
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &actual))
	if len(actual.MalwareTypes) == 0 {
		actual.MalwareTypes = nil
	}

	return &actual
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	if body == nil {
		return httptest.NewRequestWithContext(t.Context(), method, path, nil)
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	return req
}

func newHandler(t *testing.T, service *mocks.MockThreatService) http.Handler {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	handler := threatrest.NewHandler(logger, service)
	server, err := generated.NewServer(handler)
	require.NoError(t, err)

	return server
}

func expectServiceError(service *mocks.MockThreatService, method string, value *threat.Threat, err error) {
	switch method {
	case http.MethodPost:
		service.EXPECT().Create(mock.Anything, value).Return(err).Once()
	case http.MethodGet:
		service.EXPECT().Get(mock.Anything, value.MD5).Return(nil, err).Once()
	case http.MethodPut:
		service.EXPECT().Update(mock.Anything, value).Return(err).Once()
	}
}
