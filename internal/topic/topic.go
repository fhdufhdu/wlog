package topic

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fhdufhdu/wlog/internal/apperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Topic struct {
	ID                   uuid.UUID
	Name                 string
	CreatedAt, UpdatedAt time.Time
}
type Summary struct {
	ID                   uuid.UUID
	Name                 string
	PostCount, TempCount int64
}
type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool} }
func (r *Repository) List(ctx context.Context) ([]Topic, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,name,created_at,updated_at FROM topics ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Topic{}
	for rows.Next() {
		var t Topic
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
func (r *Repository) ListWithCounts(ctx context.Context) ([]Summary, error) {
	rows, err := r.pool.Query(ctx, `SELECT t.id,t.name,(SELECT COUNT(*) FROM posts p WHERE p.topic_id=t.id),(SELECT COUNT(*) FROM temp_posts tp WHERE tp.topic_id=t.id) FROM topics t ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Summary{}
	for rows.Next() {
		var t Summary
		if err := rows.Scan(&t.ID, &t.Name, &t.PostCount, &t.TempCount); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
func (r *Repository) Create(ctx context.Context, name string) (Topic, error) {
	var t Topic
	err := r.pool.QueryRow(ctx, `INSERT INTO topics(id,name)VALUES($1,$2)RETURNING id,name,created_at,updated_at`, uuid.New(), name).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}
func (r *Repository) Update(ctx context.Context, id uuid.UUID, name string) (Topic, error) {
	var t Topic
	err := r.pool.QueryRow(ctx, `UPDATE topics SET name=$2,updated_at=$3 WHERE id=$1 RETURNING id,name,created_at,updated_at`, id, name, time.Now().UTC()).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM topics WHERE id=$1`, id)
	return err == nil && tag.RowsAffected() == 1, err
}

type Service struct{ repository *Repository }

func NewService(r *Repository) *Service                      { return &Service{r} }
func (s *Service) List(ctx context.Context) ([]Topic, error) { return s.repository.List(ctx) }
func (s *Service) ListWithCounts(ctx context.Context) ([]Summary, error) {
	return s.repository.ListWithCounts(ctx)
}
func validate(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 40 {
		return "", apperr.New(apperr.Validation, "주제 이름은 1–40자로 입력해주세요.")
	}
	return name, nil
}
func topicError(err error) error {
	switch apperr.PGCode(err) {
	case "23505":
		return apperr.New(apperr.Conflict, "이미 사용 중인 주제 이름입니다.")
	case "23503":
		return apperr.New(apperr.Conflict, "글이나 임시글에서 사용 중인 주제는 제거할 수 없습니다.")
	}
	return err
}
func (s *Service) Create(ctx context.Context, name string) (Topic, error) {
	value, err := validate(name)
	if err != nil {
		return Topic{}, err
	}
	t, err := s.repository.Create(ctx, value)
	return t, topicError(err)
}
func (s *Service) Update(ctx context.Context, id uuid.UUID, name string) (Topic, error) {
	value, err := validate(name)
	if err != nil {
		return Topic{}, err
	}
	t, err := s.repository.Update(ctx, id, value)
	if errors.Is(err, pgx.ErrNoRows) {
		return Topic{}, apperr.New(apperr.NotFound, "")
	}
	return t, topicError(err)
}
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	ok, err := s.repository.Delete(ctx, id)
	if err != nil {
		return topicError(err)
	}
	if !ok {
		return apperr.New(apperr.NotFound, "")
	}
	return nil
}
