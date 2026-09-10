package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func Migrate(databaseURL string, baselineExisting bool) error {
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("migration 초기화: %w", err)
	}
	defer m.Close()
	if baselineExisting {
		if _, _, versionErr := m.Version(); errors.Is(versionErr, migrate.ErrNilVersion) {
			if err := validateCurrentSchema(context.Background(), databaseURL); err != nil {
				return fmt.Errorf("기존 DB baseline 검증: %w", err)
			}
			if err := m.Force(1); err != nil {
				return fmt.Errorf("기존 DB baseline 기록: %w", err)
			}
		}
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration 적용: %w", err)
	}
	return nil
}

func validateCurrentSchema(ctx context.Context, databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	checks := []string{
		`SELECT p.id,p.title,p.slug,p.description,p.content_markdown,p.topic_id,t.name,p.published_at,p.created_at,p.updated_at FROM posts p JOIN topics t ON t.id=p.topic_id LIMIT 0`,
		`SELECT id,post_id,title,slug,description,description_manual,content_markdown,topic_id,created_at,updated_at FROM temp_posts LIMIT 0`,
		`SELECT singleton,title,content_markdown,updated_at FROM about_page LIMIT 0`,
		`SELECT id,post_id,storage_name,original_name,mime_type,byte_size,created_at,updated_at FROM images LIMIT 0`,
	}
	for _, query := range checks {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}
