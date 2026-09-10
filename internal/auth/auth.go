package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
)

const cookieName = "wlog_session"

type Auth struct {
	username, passwordHash string
	key                    []byte
	secure                 bool
}
type Session struct{ CSRF string }

func New(username, passwordHash, secret string, secure bool) (*Auth, error) {
	if _, _, _, err := argon2id.DecodeHash(passwordHash); err != nil {
		return nil, fmt.Errorf("ADMIN_PASSWORD_HASH가 유효한 PHC 문자열이 아닙니다: %w", err)
	}
	return &Auth{username, passwordHash, []byte(secret), secure}, nil
}
func (a *Auth) Verify(username, password string) bool {
	ok, err := argon2id.ComparePasswordAndHash(password, a.passwordHash)
	return err == nil && ok && hmac.Equal([]byte(username), []byte(a.username))
}
func (a *Auth) Login(w http.ResponseWriter) error {
	csrfBytes := make([]byte, 24)
	if _, err := rand.Read(csrfBytes); err != nil {
		return err
	}
	payload := a.username + "|" + strconv.FormatInt(time.Now().Add(7*24*time.Hour).Unix(), 10) + "|" + base64.RawURLEncoding.EncodeToString(csrfBytes)
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + a.sign(payload)
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode, MaxAge: 7 * 24 * 60 * 60})
	return nil
}
func (a *Auth) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}
func (a *Auth) Session(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, false
	}
	pieces := strings.Split(cookie.Value, ".")
	if len(pieces) != 2 {
		return Session{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(pieces[0])
	if err != nil || !hmac.Equal([]byte(pieces[1]), []byte(a.sign(string(raw)))) {
		return Session{}, false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || parts[0] != a.username {
		return Session{}, false
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || expires < time.Now().Unix() {
		return Session{}, false
	}
	return Session{CSRF: parts[2]}, true
}
func (a *Auth) sign(value string) string {
	mac := hmac.New(sha256.New, a.key)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
