package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"recibosgo/internal/models"
)

type ctxKey string

const (
	ctxUser    ctxKey = "user"
	cookieName string = "recibosgo_session"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

type AppCtx struct {
	DB     *gorm.DB
	Secret string
}

func New(db *gorm.DB, secret string) *AppCtx {
	return &AppCtx{DB: db, Secret: secret}
}

func (a *AppCtx) sign(data string) string {
	mac := hmac.New(sha256.New, []byte(a.Secret))
	mac.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *AppCtx) SetSession(w http.ResponseWriter, userID uint) {
	raw := fmt.Sprintf("%d:%s", userID, a.sign(strconv.FormatUint(uint64(userID), 10)))
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 30,
	})
}

func (a *AppCtx) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func (a *AppCtx) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		parts := strings.Split(cookie.Value, ":")
		if len(parts) != 2 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		userID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if parts[1] != a.sign(parts[0]) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		var user models.User
		if err := a.DB.First(&user, userID).Error; err != nil {
			a.ClearSession(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUser, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Acceso denegado", http.StatusForbidden)
		})
	}
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(ctxUser).(*models.User)
	return user, ok
}
