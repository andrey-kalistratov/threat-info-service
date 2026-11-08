// Package rest реализует HTTP-интерфейс сведений об угрозах.
package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/andrey-kalistratov/threat-info-service/internal/logging"
	"github.com/andrey-kalistratov/threat-info-service/internal/rest/generated"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
)

// Handler обрабатывает HTTP-операции со сведениями об угрозах.
type Handler struct {
	logger  logging.Logger
	service threatService
}

type threatService interface {
	Create(context.Context, *threat.Threat) error
	Get(context.Context, threat.MD5) (*threat.Threat, error)
	Update(context.Context, *threat.Threat) error
}

// NewHandler создаёт предметный HTTP-обработчик с готовыми зависимостями.
func NewHandler(logger logging.Logger, service threatService) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
	}
}

// CreateThreat создаёт запись и возвращает сохранённые сведения.
func (h *Handler) CreateThreat(ctx context.Context, body *generated.Threat) (generated.CreateThreatRes, error) {
	value, err := threatFromCreateRequest(body)
	if err != nil {
		return &generated.CreateThreatBadRequest{
			Message: generated.NewOptString(err.Error()),
		}, nil
	}

	if err := h.service.Create(ctx, value); err != nil {
		if errors.Is(err, threat.ErrAlreadyExists) {
			return &generated.CreateThreatConflict{
				Message: generated.NewOptString("Threat already exists"),
			}, nil
		}

		h.logger.ErrorContext(ctx, "CreateThreat", "error", err)
		return &generated.CreateThreatInternalServerError{
			Message: generated.NewOptString(http.StatusText(http.StatusInternalServerError)),
		}, nil
	}

	return threatToResponse(value), nil
}

// GetInfoByMd5 возвращает сведения по MD5 из пути запроса.
func (h *Handler) GetInfoByMd5(ctx context.Context, params generated.GetInfoByMd5Params) (generated.GetInfoByMd5Res, error) {
	md5, err := md5FromRequest(params.MD5)
	if err != nil {
		return &generated.GetInfoByMd5BadRequest{
			Message: generated.NewOptString(err.Error()),
		}, nil
	}

	value, err := h.service.Get(ctx, md5)
	if err != nil {
		if errors.Is(err, threat.ErrNotFound) {
			return &generated.GetInfoByMd5NotFound{
				Message: generated.NewOptString("Threat not found"),
			}, nil
		}

		h.logger.ErrorContext(ctx, "GetInfoByMd5", "error", err)
		return &generated.GetInfoByMd5InternalServerError{
			Message: generated.NewOptString(http.StatusText(http.StatusInternalServerError)),
		}, nil
	}

	return threatToResponse(value), nil
}

// UpdateThreat полностью заменяет сведения, сохраняя MD5 из пути запроса.
func (h *Handler) UpdateThreat(ctx context.Context, body *generated.ThreatData, params generated.UpdateThreatParams) (generated.UpdateThreatRes, error) {
	value, err := threatFromUpdateRequest(body, params)
	if err != nil {
		return &generated.UpdateThreatBadRequest{
			Message: generated.NewOptString(err.Error()),
		}, nil
	}

	if err := h.service.Update(ctx, value); err != nil {
		if errors.Is(err, threat.ErrNotFound) {
			return &generated.UpdateThreatNotFound{
				Message: generated.NewOptString("Threat not found"),
			}, nil
		}

		h.logger.ErrorContext(ctx, "UpdateThreat", "error", err)
		return &generated.UpdateThreatInternalServerError{
			Message: generated.NewOptString(http.StatusText(http.StatusInternalServerError)),
		}, nil
	}

	return threatToResponse(value), nil
}
