package threat

import "context"

// Service выполняет операции со сведениями об угрозах.
type Service struct {
	repository Repository
}

// Repository — хранилище, необходимое сервису.
type Repository interface {
	Create(context.Context, *Threat) error
	Get(context.Context, MD5) (*Threat, error)
	Update(context.Context, *Threat) error
}

// NewService создаёт сервис с заданным хранилищем.
func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

// Create сохраняет новую угрозу.
func (s *Service) Create(ctx context.Context, value *Threat) error {
	return s.repository.Create(ctx, value)
}

// Get возвращает угрозу по MD5.
func (s *Service) Get(ctx context.Context, md5 MD5) (*Threat, error) {
	return s.repository.Get(ctx, md5)
}

// Update полностью заменяет сведения существующей угрозы.
func (s *Service) Update(ctx context.Context, value *Threat) error {
	return s.repository.Update(ctx, value)
}
