package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/andrey-kalistratov/threat-info-service/internal/logging"
	"github.com/andrey-kalistratov/threat-info-service/internal/rest/generated"
)

type transportErrorHandler struct {
	logger logging.Logger
}

func (h *transportErrorHandler) Handle(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	var (
		decodeRequest *ogenerrors.DecodeRequestError
		decodeParams  *ogenerrors.DecodeParamsError
	)
	if errors.As(err, &decodeRequest) || errors.As(err, &decodeParams) {
		h.writeError(ctx, w, http.StatusBadRequest)
		return
	}

	// Сгенерированный JSON-путь уже начал ответ до ошибки его отправки.
	h.logger.ErrorContext(ctx, "Send HTTP response", "error", err)
}

func (h *transportErrorHandler) writeError(ctx context.Context, w http.ResponseWriter, status int) {
	response := &generated.ErrorResponse{
		Message: generated.NewOptString(http.StatusText(status)),
	}
	encoder := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := encoder.Encode(response); err != nil {
		h.logger.ErrorContext(ctx, "Send HTTP error response", "error", err)
	}
}
