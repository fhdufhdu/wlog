# Wlog

Go 1.26, `net/http`, pgx, PostgreSQL로 만든 서버 렌더링 개인 블로그입니다. HTTP 계층은 `AppHandler → AppMiddleware → Controller → AppMux` 구조의 작은 자체 프레임워크로 구성했습니다.

## 로컬 실행

1. `docker compose up -d postgres`로 PostgreSQL을 시작합니다.
2. `cp .env.example .env`로 환경 파일을 만듭니다.
3. `go run ./cmd/hash_password`와 `openssl rand -base64 48`의 결과를 `.env`의 `ADMIN_PASSWORD_HASH`, `SESSION_SECRET`에 넣습니다.
4. `go run ./cmd/wlog`로 실행합니다. 시작할 때 `golang-migrate` migration이 자동 적용됩니다.
5. 공개 블로그는 `http://127.0.0.1:3000`, 글 관리는 `/admin`입니다.

## 구조

- `cmd/wlog`: 의존성 조립, 서버 시작과 graceful shutdown
- `internal/app`: AppHandler, AppMiddleware, Controller, AppMux 프레임워크
- `internal/web`: 공개·관리자 HTTP 컨트롤러와 템플릿 모델
- `internal/post`, `topic`, `about`, `image`: 도메인 서비스와 PostgreSQL repository
- `migrations`: 현재 스키마 기준 `golang-migrate` migration
- `templates`, 정적 파일: 서버 렌더링 화면과 관리자 편집기

새 DB는 최신 스키마로 바로 생성됩니다. 기존 Rust/SQLx DB를 유지하는 컨테이너 배포에서는 `MIGRATION_BASELINE_EXISTING=true`를 설정하세요. 앱이 현재 스키마의 필수 테이블과 컬럼을 검증하고 `golang-migrate` 버전 1로 baseline한 뒤 정상 기동합니다. 이미 baseline된 DB에는 이 설정이 영향을 주지 않습니다.

## 제공 기능

- 공개 글 목록, 주제 필터, 글 상세와 이전·다음 글
- Argon2id 관리자 로그인, HMAC 서명 세션 쿠키, CSRF 검증
- 글·임시글·주제·소개 관리와 자동 임시저장
- 이미지 업로드, WebP 변환·리사이징, SVG 정화, 미사용 이미지 정리
- Markdown 렌더링과 HTML 정화, 코드 강조
- canonical, Open Graph, JSON-LD, `robots.txt`, `sitemap.xml`
- live/readiness 상태 확인과 graceful shutdown

## 검증

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
go build ./cmd/wlog
```

## Docker 이미지 발행

`1.2.3` 형식의 Git 태그를 푸시하면 GitHub Actions가 테스트와 정적 분석 후 `ghcr.io/fhdufhdu/wlog` 이미지를 발행합니다.
