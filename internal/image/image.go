package image

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	stdimage "image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/fhdufhdu/wlog/internal/apperr"
	markdownutil "github.com/fhdufhdu/wlog/internal/markdown"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

type orphan struct {
	ID   uuid.UUID
	Name string
}

func (r *Repository) Create(ctx context.Context, id uuid.UUID, name, original, mime string, size int64) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO images(id,storage_name,original_name,mime_type,byte_size)VALUES($1,$2,$3,$4,$5)`, id, name, original, mime, size)
	return err
}
func (r *Repository) Orphans(ctx context.Context, before time.Time) ([]orphan, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,storage_name FROM images WHERE post_id IS NULL AND updated_at<$1`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []orphan
	for rows.Next() {
		var o orphan
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (r *Repository) AllMarkdown(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT content_markdown FROM posts UNION ALL SELECT content_markdown FROM temp_posts UNION ALL SELECT content_markdown FROM about_page`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *Repository) DeleteOrphan(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM images WHERE id=$1 AND post_id IS NULL`, id)
	return err == nil && tag.RowsAffected() == 1, err
}

type Service struct {
	repository *Repository
	dir        string
	grace      time.Duration
}

func NewService(r *Repository, dir string, hours int) *Service {
	return &Service{r, dir, time.Duration(hours) * time.Hour}
}
func (s *Service) Register(ctx context.Context, id uuid.UUID, name, original, mime string, size int64) error {
	original = strings.TrimSpace(original)
	if original == "" {
		original = name
	}
	runes := []rune(original)
	if len(runes) > 255 {
		original = string(runes[:255])
	}
	return s.repository.Create(ctx, id, name, original, mime, size)
}
func (s *Service) Cleanup(ctx context.Context) (int, error) {
	items, err := s.repository.Orphans(ctx, time.Now().Add(-s.grace))
	if err != nil {
		return 0, err
	}
	content, err := s.repository.AllMarkdown(ctx)
	if err != nil {
		return 0, err
	}
	used := map[string]bool{}
	for _, text := range content {
		for _, name := range markdownutil.UploadNames(text) {
			used[name] = true
		}
	}
	removed := 0
	for _, item := range items {
		if used[item.Name] {
			continue
		}
		err := os.Remove(filepath.Join(s.dir, item.Name))
		if err != nil && !os.IsNotExist(err) {
			slog.Warn("orphan image removal failed", "file", item.Name, "error", err)
			continue
		}
		ok, err := s.repository.DeleteOrphan(ctx, item.ID)
		if err != nil {
			return removed, err
		}
		if ok {
			removed++
		}
	}
	return removed, nil
}

type Normalized struct {
	Bytes           []byte
	Mime, Extension string
}

func NormalizeRaster(data []byte, mime string, maxDimension int, maxPixels int64, quality float32) (Normalized, error) {
	cfg, format, err := stdimage.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Normalized{}, apperr.New(apperr.Validation, "손상되었거나 지원하지 않는 이미지 파일입니다.")
	}
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if cfg.Width <= 0 || cfg.Height <= 0 || pixels > maxPixels {
		return Normalized{}, apperr.New(apperr.Validation, fmt.Sprintf("이미지 해상도가 너무 큽니다. 전체 픽셀 수는 %d 이하여야 합니다.", maxPixels))
	}
	if mime == "image/gif" || format == "gif" {
		if cfg.Width > maxDimension || cfg.Height > maxDimension {
			return Normalized{}, apperr.New(apperr.Validation, fmt.Sprintf("움직이는 GIF는 자동 축소하지 않습니다. 가로와 세로를 각각 %dpx 이하로 줄여주세요.", maxDimension))
		}
		if _, err := gif.DecodeAll(bytes.NewReader(data)); err != nil {
			return Normalized{}, apperr.New(apperr.Validation, "손상되었거나 지원하지 않는 이미지 파일입니다.")
		}
		return Normalized{data, "image/gif", "gif"}, nil
	}
	img, _, err := stdimage.Decode(bytes.NewReader(data))
	if err != nil {
		return Normalized{}, apperr.New(apperr.Validation, "손상되었거나 지원하지 않는 이미지 파일입니다.")
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxDimension || h > maxDimension {
		ratio := float64(maxDimension) / float64(max(w, h))
		nw, nh := max(1, int(float64(w)*ratio)), max(1, int(float64(h)*ratio))
		dst := stdimage.NewRGBA(stdimage.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
	}
	var out bytes.Buffer
	if err := webp.Encode(&out, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		return Normalized{}, apperr.New(apperr.Validation, "이미지를 WebP로 변환하지 못했습니다.")
	}
	return Normalized{out.Bytes(), "image/webp", "webp"}, nil
}

func SanitizeSVG(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	enc := xml.NewEncoder(&out)
	blocked := 0
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apperr.New(apperr.Validation, "올바르고 안전한 SVG 파일인지 확인해주세요.")
		}
		switch value := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(value.Name.Local)
			if blocked > 0 || name == "script" || name == "foreignobject" || name == "iframe" {
				blocked++
				continue
			}
			attrs := value.Attr[:0]
			for _, attr := range value.Attr {
				key := strings.ToLower(attr.Name.Local)
				val := strings.TrimSpace(strings.ToLower(attr.Value))
				if strings.HasPrefix(key, "on") || key == "href" || key == "src" || strings.Contains(val, "javascript:") || strings.Contains(val, "data:text/html") {
					continue
				}
				attrs = append(attrs, attr)
			}
			value.Attr = attrs
			if err := enc.EncodeToken(value); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if blocked > 0 {
				blocked--
				continue
			}
			if err := enc.EncodeToken(value); err != nil {
				return nil, err
			}
		default:
			if blocked == 0 {
				if err := enc.EncodeToken(token); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	if out.Len() == 0 || !bytes.Contains(bytes.ToLower(out.Bytes()), []byte("<svg")) {
		return nil, apperr.New(apperr.Validation, "내용이 없는 SVG 파일은 업로드할 수 없습니다.")
	}
	return out.Bytes(), nil
}
