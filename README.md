# Wlog

Rust 2024, Axum, SQLx, PostgreSQL로 만든 서버 렌더링 개인 블로그입니다. 공개 페이지는 JavaScript 없이 읽을 수 있고, 관리자 화면에서 Markdown 글과 이미지를 관리합니다.

## 로컬 실행

1. `docker compose up -d postgres`로 PostgreSQL을 시작합니다.
2. `cp .env.example .env`로 환경 파일을 만듭니다.
3. `cargo run --bin hash_password`와 `openssl rand -base64 48`의 결과를 `.env`의 `ADMIN_PASSWORD_HASH`, `SESSION_SECRET`에 넣습니다. 운영에서는 배포 서비스의 Secret 환경변수를 사용하세요.
4. `cargo run --bin wlog`로 실행합니다. 시작할 때 migration이 자동 적용됩니다.
5. 공개 블로그는 `http://127.0.0.1:3000`, 글 관리는 `/admin`입니다.

## 제공 기능

- 공개 글 목록과 분류 필터, 개별 글의 서버 렌더링 HTML
- 환경변수의 관리자 아이디와 Argon2id 비밀번호 해시
- `axum-extra` 서명 쿠키, `HttpOnly`, `SameSite=Strict` 세션과 CSRF 토큰
- 권한 확인이 적용된 글 생성·수정·삭제와 이미지 업로드
- 업로드 이미지 DB 기록, 래스터 이미지 WebP 정규화·리사이징, 게시글 연결 동기화, 미사용 파일 자동 정리
- 공개 글과 분리된 `temp_posts` 자동 임시저장, 브라우저 실시간 미리보기, 명시적 발행
- DB에는 Markdown 원문만 보관하고 공개 요청에서 안전한 HTML로 서버 렌더링
- CommonMark/GFM Markdown, 안전한 HTML 정리, 서버 측 코드 구문 강조
- 본문 기반 SEO 설명 자동 생성, canonical, Open Graph, Twitter Card, BlogPosting JSON-LD
- `robots.txt`, `sitemap.xml`, live/readiness 상태 확인

## 운영 확인

- HTTPS 환경에서는 `PUBLIC_BASE_URL=https://...`, `SECURE_COOKIE=true`로 둡니다.
- `uploads`는 영속 볼륨 또는 오브젝트 스토리지로 보존해야 합니다.
- 래스터 이미지는 긴 변 기준 `IMAGE_MAX_DIMENSION` 이하로 축소되고 `IMAGE_WEBP_QUALITY` 품질의 WebP로 저장됩니다. GIF와 정화된 SVG는 원본 형식을 유지합니다.
- 압축 해제 시 과도한 메모리 사용을 막는 `IMAGE_MAX_PIXELS`는 업로드 용량 제한인 `MAX_UPLOAD_BYTES`와 별개입니다.
- 본문·임시글·소개에서 참조하지 않는 이미지는 `IMAGE_ORPHAN_GRACE_HOURS` 이후 주기적으로 정리됩니다.
- 애플리케이션과 PostgreSQL은 TLS 프록시 뒤에서 실행합니다.
- DB와 업로드 파일을 함께 백업합니다.

## Docker 이미지 발행

`1.2.3` 형식의 Git 태그를 푸시하면 GitHub Actions가 테스트와 release 빌드를
Ubuntu 호스트에서 실행한 뒤 `ghcr.io/fhdufhdu/wlog`에 이미지를 발행합니다.
Docker 빌드는 미리 빌드된 바이너리와 정적 파일만 런타임 이미지에 복사합니다.

```sh
git tag 0.1.0
git push origin 0.1.0
```

```sh
cargo fmt --all -- --check
cargo check --all-targets
cargo test --all-targets
cargo clippy --all-targets -- -D warnings
```
