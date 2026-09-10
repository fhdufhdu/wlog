package about

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fhdufhdu/wlog/internal/apperr"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Page struct {
	Title, ContentMarkdown string
	UpdatedAt              time.Time
}
type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool} }
func (r *Repository) Get(ctx context.Context) (Page, error) {
	var p Page
	err := r.pool.QueryRow(ctx, `SELECT title,content_markdown,updated_at FROM about_page WHERE singleton=TRUE`).Scan(&p.Title, &p.ContentMarkdown, &p.UpdatedAt)
	return p, err
}
func (r *Repository) Update(ctx context.Context, title, content string) (Page, error) {
	var p Page
	err := r.pool.QueryRow(ctx, `UPDATE about_page SET title=$1,content_markdown=$2,updated_at=$3 WHERE singleton=TRUE RETURNING title,content_markdown,updated_at`, title, content, time.Now().UTC()).Scan(&p.Title, &p.ContentMarkdown, &p.UpdatedAt)
	return p, err
}

type Service struct{ repository *Repository }

func NewService(r *Repository) *Service                  { return &Service{r} }
func (s *Service) Get(ctx context.Context) (Page, error) { return s.repository.Get(ctx) }
func (s *Service) Update(ctx context.Context, title, content string) (Page, error) {
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > 120 {
		return Page{}, apperr.New(apperr.Validation, "소개 제목은 1–120자로 입력해주세요.")
	}
	return s.repository.Update(ctx, title, content)
}
