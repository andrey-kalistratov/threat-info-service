// Package postgres хранит сведения об угрозах в PostgreSQL.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
)

// Repository выполняет операции со сведениями об угрозах.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository использует готовый пул. Его жизненным циклом управляет вызывающий код.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create сохраняет новую угрозу. Повторившийся MD5 даёт [threat.ErrAlreadyExists].
func (r *Repository) Create(ctx context.Context, value *threat.Threat) error {
	const query = `
		INSERT INTO threats (md5, name, description, malware_types, confidence)
		VALUES ($1, $2, $3, $4, $5)`

	prepared := prepareThreatForSave(value)
	_, err := r.pool.Exec(ctx, query,
		prepared.MD5, prepared.Name, prepared.Description, prepared.MalwareTypes, prepared.Confidence,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == "threats_pkey" {
		return threat.ErrAlreadyExists
	}

	return err
}

// Get возвращает угрозу по MD5 или [threat.ErrNotFound].
func (r *Repository) Get(ctx context.Context, md5 threat.MD5) (*threat.Threat, error) {
	const query = `
		SELECT md5, name, description, malware_types, confidence
		FROM threats
		WHERE md5 = $1`

	var value threat.Threat

	err := r.pool.QueryRow(ctx, query, md5).Scan(
		&value.MD5, &value.Name, &value.Description, &value.MalwareTypes, &value.Confidence,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, threat.ErrNotFound
		}

		return nil, err
	}

	return &value, nil
}

// Update полностью заменяет поля существующей угрозы, сохраняя её MD5.
// Если запись отсутствует, возвращает [threat.ErrNotFound].
func (r *Repository) Update(ctx context.Context, value *threat.Threat) error {
	const query = `
		UPDATE threats
		SET name = $2, description = $3, malware_types = $4, confidence = $5
		WHERE md5 = $1`

	prepared := prepareThreatForSave(value)
	result, err := r.pool.Exec(ctx, query,
		prepared.MD5, prepared.Name, prepared.Description, prepared.MalwareTypes, prepared.Confidence,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return threat.ErrNotFound
	}

	return nil
}

// prepareThreatForSave готовит сущность для записи, не изменяя входную модель.
// Если список категорий nil, создаёт копию с пустым срезом для записи массива вместо NULL.
func prepareThreatForSave(value *threat.Threat) *threat.Threat {
	if value.MalwareTypes != nil {
		return value
	}

	prepared := *value
	prepared.MalwareTypes = []threat.MalwareType{}
	return &prepared
}
