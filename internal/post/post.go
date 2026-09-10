package post

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fhdufhdu/wlog/internal/apperr"
	markdownutil "github.com/fhdufhdu/wlog/internal/markdown"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Post struct {
	ID                                        uuid.UUID
	Title, Slug, Description, ContentMarkdown string
	TopicID                                   uuid.UUID
	TopicName                                 string
	PublishedAt, CreatedAt, UpdatedAt         time.Time
}
type ListItem struct {
	ID                                  uuid.UUID
	Title, Slug, Description, TopicName string
	PublishedAt, UpdatedAt              time.Time
}
type TempListItem struct {
	ID          uuid.UUID
	Title, Slug string
	UpdatedAt   time.Time
}
type Link struct{ Title, Slug string }
type Neighbors struct{ Previous, Next *Link }
type TempPost struct {
	ID                       uuid.UUID
	PostID                   *uuid.UUID
	Title, Slug, Description string
	DescriptionManual        bool
	ContentMarkdown          string
	TopicID                  *uuid.UUID
	TopicName                *string
	CreatedAt, UpdatedAt     time.Time
}
type Form struct {
	Title, Slug, Description, TopicID, ContentMarkdown, CSRFToken string
	DescriptionManual                                             bool
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

func scanPost(row pgx.Row) (Post, error) {
	var p Post
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.Description, &p.ContentMarkdown, &p.TopicID, &p.TopicName, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
func scanTemp(row pgx.Row) (TempPost, error) {
	var p TempPost
	err := row.Scan(&p.ID, &p.PostID, &p.Title, &p.Slug, &p.Description, &p.DescriptionManual, &p.ContentMarkdown, &p.TopicID, &p.TopicName, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) ListPublic(ctx context.Context, topicID *uuid.UUID) ([]ListItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.id,p.title,p.slug,p.description,t.name,p.published_at,p.updated_at FROM posts p JOIN topics t ON t.id=p.topic_id WHERE ($1::UUID IS NULL OR p.topic_id=$1) ORDER BY p.published_at DESC,p.id DESC LIMIT 100`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListItem{}
	for rows.Next() {
		var p ListItem
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Description, &p.TopicName, &p.PublishedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
func (r *Repository) ListAll(ctx context.Context) ([]ListItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.id,p.title,p.slug,p.description,t.name,p.published_at,p.updated_at FROM posts p JOIN topics t ON t.id=p.topic_id ORDER BY p.published_at DESC,p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListItem{}
	for rows.Next() {
		var p ListItem
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Description, &p.TopicName, &p.PublishedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
func (r *Repository) ListUnlinkedTemp(ctx context.Context) ([]TempListItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,title,slug,updated_at FROM temp_posts WHERE post_id IS NULL ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TempListItem{}
	for rows.Next() {
		var p TempListItem
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
func (r *Repository) FindPublicSlug(ctx context.Context, value string) (Post, error) {
	return scanPost(r.pool.QueryRow(ctx, `SELECT p.id,p.title,p.slug,p.description,p.content_markdown,p.topic_id,t.name,p.published_at,p.created_at,p.updated_at FROM posts p JOIN topics t ON t.id=p.topic_id WHERE p.slug=$1`, value))
}
func (r *Repository) FindID(ctx context.Context, id uuid.UUID) (Post, error) {
	return scanPost(r.pool.QueryRow(ctx, `SELECT p.id,p.title,p.slug,p.description,p.content_markdown,p.topic_id,t.name,p.published_at,p.created_at,p.updated_at FROM posts p JOIN topics t ON t.id=p.topic_id WHERE p.id=$1`, id))
}
func (r *Repository) FindTempID(ctx context.Context, id uuid.UUID) (TempPost, error) {
	return scanTemp(r.pool.QueryRow(ctx, `SELECT tp.id,tp.post_id,tp.title,tp.slug,tp.description,tp.description_manual,tp.content_markdown,tp.topic_id,t.name,tp.created_at,tp.updated_at FROM temp_posts tp LEFT JOIN topics t ON t.id=tp.topic_id WHERE tp.id=$1`, id))
}
func (r *Repository) CreateEmptyTemp(ctx context.Context) (TempPost, error) {
	return scanTemp(r.pool.QueryRow(ctx, `INSERT INTO temp_posts(id) VALUES($1) RETURNING id,post_id,title,slug,description,description_manual,content_markdown,topic_id,NULL::VARCHAR,created_at,updated_at`, uuid.New()))
}
func (r *Repository) TempForPost(ctx context.Context, postID uuid.UUID) (TempPost, error) {
	return scanTemp(r.pool.QueryRow(ctx, `WITH u AS (INSERT INTO temp_posts(id,post_id,title,slug,description,description_manual,content_markdown,topic_id) SELECT $1,id,title,slug,description,TRUE,content_markdown,topic_id FROM posts WHERE id=$2 ON CONFLICT(post_id) DO UPDATE SET post_id=EXCLUDED.post_id RETURNING id,post_id,title,slug,description,description_manual,content_markdown,topic_id,created_at,updated_at) SELECT u.id,u.post_id,u.title,u.slug,u.description,u.description_manual,u.content_markdown,u.topic_id,t.name,u.created_at,u.updated_at FROM u LEFT JOIN topics t ON t.id=u.topic_id`, uuid.New(), postID))
}
func (r *Repository) UpdateTemp(ctx context.Context, p TempPost) (TempPost, error) {
	return scanTemp(r.pool.QueryRow(ctx, `WITH u AS (UPDATE temp_posts SET title=$2,slug=$3,description=$4,description_manual=$5,content_markdown=$6,topic_id=$7,updated_at=$8 WHERE id=$1 RETURNING id,post_id,title,slug,description,description_manual,content_markdown,topic_id,created_at,updated_at) SELECT u.id,u.post_id,u.title,u.slug,u.description,u.description_manual,u.content_markdown,u.topic_id,t.name,u.created_at,u.updated_at FROM u LEFT JOIN topics t ON t.id=u.topic_id`, p.ID, p.Title, p.Slug, p.Description, p.DescriptionManual, p.ContentMarkdown, p.TopicID, p.UpdatedAt))
}
func (r *Repository) FindNeighbors(ctx context.Context, p Post, topicID *uuid.UUID) (Neighbors, error) {
	var pt, ps, nt, ns *string
	err := r.pool.QueryRow(ctx, `SELECT previous.title,previous.slug,next.title,next.slug FROM (VALUES($1::TIMESTAMPTZ,$2::UUID,$3::UUID)) current(published_at,id,topic_id) LEFT JOIN LATERAL(SELECT title,slug FROM posts WHERE(current.topic_id IS NULL OR topic_id=current.topic_id)AND(published_at,id)<(current.published_at,current.id)ORDER BY published_at DESC,id DESC LIMIT 1)previous ON TRUE LEFT JOIN LATERAL(SELECT title,slug FROM posts WHERE(current.topic_id IS NULL OR topic_id=current.topic_id)AND(published_at,id)>(current.published_at,current.id)ORDER BY published_at ASC,id ASC LIMIT 1)next ON TRUE`, p.PublishedAt, p.ID, topicID).Scan(&pt, &ps, &nt, &ns)
	if err != nil {
		return Neighbors{}, err
	}
	n := Neighbors{}
	if pt != nil && ps != nil {
		n.Previous = &Link{*pt, *ps}
	}
	if nt != nil && ns != nil {
		n.Next = &Link{*nt, *ns}
	}
	return n, nil
}
func (r *Repository) Publish(ctx context.Context, p TempPost, names []string) (Post, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Post{}, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	var out Post
	if p.PostID != nil {
		out, err = scanPost(tx.QueryRow(ctx, `WITH u AS(UPDATE posts SET title=$2,slug=$3,description=$4,content_markdown=$5,topic_id=$6,updated_at=$7 WHERE id=$1 RETURNING id,title,slug,description,content_markdown,topic_id,published_at,created_at,updated_at)SELECT u.id,u.title,u.slug,u.description,u.content_markdown,u.topic_id,t.name,u.published_at,u.created_at,u.updated_at FROM u JOIN topics t ON t.id=u.topic_id`, *p.PostID, p.Title, p.Slug, p.Description, p.ContentMarkdown, p.TopicID, now))
	} else {
		out, err = scanPost(tx.QueryRow(ctx, `WITH i AS(INSERT INTO posts(id,title,slug,description,content_markdown,topic_id,published_at,created_at,updated_at)VALUES($1,$2,$3,$4,$5,$6,$7,$7,$7)RETURNING id,title,slug,description,content_markdown,topic_id,published_at,created_at,updated_at)SELECT i.id,i.title,i.slug,i.description,i.content_markdown,i.topic_id,t.name,i.published_at,i.created_at,i.updated_at FROM i JOIN topics t ON t.id=i.topic_id`, uuid.New(), p.Title, p.Slug, p.Description, p.ContentMarkdown, p.TopicID, now))
	}
	if err != nil {
		return Post{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE images SET post_id=NULL,updated_at=$3 WHERE post_id=$1 AND NOT(storage_name=ANY($2::TEXT[]))`, out.ID, names, now); err != nil {
		return Post{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE images SET post_id=$1,updated_at=$3 WHERE post_id IS NULL AND storage_name=ANY($2::TEXT[])`, out.ID, names, now); err != nil {
		return Post{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM temp_posts WHERE id=$1`, p.ID); err != nil {
		return Post{}, err
	}
	return out, tx.Commit(ctx)
}
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	if _, err = tx.Exec(ctx, `UPDATE images SET post_id=NULL,updated_at=$2 WHERE post_id=$1`, id, now); err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM posts WHERE id=$1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, tx.Commit(ctx)
}
func (r *Repository) DeleteTemp(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM temp_posts WHERE id=$1`, id)
	return err == nil && tag.RowsAffected() == 1, err
}

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository} }
func (s *Service) ListPublic(ctx context.Context, id *uuid.UUID) ([]ListItem, error) {
	return s.repository.ListPublic(ctx, id)
}
func (s *Service) ListAll(ctx context.Context) ([]ListItem, error) { return s.repository.ListAll(ctx) }
func (s *Service) ListUnlinkedTemp(ctx context.Context) ([]TempListItem, error) {
	return s.repository.ListUnlinkedTemp(ctx)
}
func notFound[T any](value T, err error) (T, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return value, apperr.New(apperr.NotFound, "")
	}
	return value, err
}
func (s *Service) PublicBySlug(ctx context.Context, slug string) (Post, error) {
	return notFound(s.repository.FindPublicSlug(ctx, slug))
}
func (s *Service) ByID(ctx context.Context, id uuid.UUID) (Post, error) {
	return notFound(s.repository.FindID(ctx, id))
}
func (s *Service) TempByID(ctx context.Context, id uuid.UUID) (TempPost, error) {
	return notFound(s.repository.FindTempID(ctx, id))
}
func (s *Service) NewTemp(ctx context.Context) (TempPost, error) {
	return s.repository.CreateEmptyTemp(ctx)
}
func (s *Service) TempForPost(ctx context.Context, id uuid.UUID) (TempPost, error) {
	return notFound(s.repository.TempForPost(ctx, id))
}
func (s *Service) Adjacent(ctx context.Context, p Post, withinTopic bool) (Neighbors, error) {
	var id *uuid.UUID
	if withinTopic {
		id = &p.TopicID
	}
	return s.repository.FindNeighbors(ctx, p, id)
}
func (s *Service) SaveTemp(ctx context.Context, id uuid.UUID, form Form) (TempPost, error) {
	clean, err := validate(form, false)
	if err != nil {
		return TempPost{}, err
	}
	old, err := s.TempByID(ctx, id)
	if err != nil {
		return TempPost{}, err
	}
	clean.ID = id
	clean.PostID = old.PostID
	clean.CreatedAt = old.CreatedAt
	clean.UpdatedAt = time.Now().UTC()
	saved, err := s.repository.UpdateTemp(ctx, clean)
	return saved, dbError(err)
}
func (s *Service) PublishTemp(ctx context.Context, id uuid.UUID, form Form) (Post, error) {
	clean, err := validate(form, true)
	if err != nil {
		return Post{}, err
	}
	old, err := s.TempByID(ctx, id)
	if err != nil {
		return Post{}, err
	}
	clean.ID = id
	clean.PostID = old.PostID
	clean.CreatedAt = old.CreatedAt
	clean.UpdatedAt = time.Now().UTC()
	saved, err := s.repository.UpdateTemp(ctx, clean)
	if err != nil {
		return Post{}, dbError(err)
	}
	published, err := s.repository.Publish(ctx, saved, markdownutil.UploadNames(saved.ContentMarkdown))
	return published, dbError(err)
}
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	ok, err := s.repository.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.NotFound, "")
	}
	return nil
}
func (s *Service) DeleteTemp(ctx context.Context, id uuid.UUID) error {
	ok, err := s.repository.DeleteTemp(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.NotFound, "")
	}
	return nil
}

func validate(form Form, publishing bool) (TempPost, error) {
	title := strings.TrimSpace(form.Title)
	content := strings.TrimSpace(form.ContentMarkdown)
	slugValue := slug.Make(strings.TrimSpace(form.Slug))
	description := strings.TrimSpace(form.Description)
	if !form.DescriptionManual {
		description = markdownutil.Excerpt(content, 80)
	}
	var topicID *uuid.UUID
	if strings.TrimSpace(form.TopicID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(form.TopicID))
		if err != nil {
			return TempPost{}, apperr.New(apperr.Validation, "올바른 주제를 선택해주세요.")
		}
		topicID = &id
	}
	if utf8.RuneCountInString(title) > 120 || (publishing && title == "") {
		return TempPost{}, apperr.New(apperr.Validation, "제목은 1–120자로 입력해주세요.")
	}
	if len(slugValue) > 160 || (publishing && slugValue == "") {
		return TempPost{}, apperr.New(apperr.Validation, "주소는 영문·숫자 중심의 1–160자로 입력해주세요.")
	}
	if utf8.RuneCountInString(description) > 200 || (publishing && description == "") {
		return TempPost{}, apperr.New(apperr.Validation, "요약은 1–200자로 입력해주세요.")
	}
	if publishing && topicID == nil {
		return TempPost{}, apperr.New(apperr.Validation, "주제를 선택해주세요.")
	}
	if publishing && content == "" {
		return TempPost{}, apperr.New(apperr.Validation, "본문을 입력해주세요.")
	}
	return TempPost{Title: title, Slug: slugValue, Description: description, DescriptionManual: form.DescriptionManual, ContentMarkdown: content, TopicID: topicID}, nil
}
func dbError(err error) error {
	switch apperr.PGCode(err) {
	case "23505":
		return apperr.New(apperr.Conflict, "이미 사용 중인 글 주소입니다.")
	case "23503":
		return apperr.New(apperr.Validation, "선택한 주제를 찾을 수 없습니다.")
	}
	return err
}
