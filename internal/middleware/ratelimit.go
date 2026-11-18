package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter limita la cantidad de requests por IP en una ventana de tiempo.
type RateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	max     int
	ventana time.Duration
}

func NewRateLimiter(max int, ventana time.Duration) *RateLimiter {
	return &RateLimiter{hits: map[string][]time.Time{}, max: max, ventana: ventana}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	ahora := time.Now()
	validos := rl.hits[ip][:0]
	for _, t := range rl.hits[ip] {
		if ahora.Sub(t) < rl.ventana {
			validos = append(validos, t)
		}
	}
	if len(validos) >= rl.max {
		rl.hits[ip] = validos
		return false
	}
	rl.hits[ip] = append(validos, ahora)
	return true
}

// Limit responde 429 si se supera el límite por IP.
func (rl *RateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(clientIP(r)) {
			http.Error(w, "Demasiados intentos. Probá más tarde.", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func clientIP(r *http.Request) string {
	// shortcut: X-Forwarded-For es spoofeable; detrás de un proxy confiable
	// conviene usar la IP del proxy para evitar evasiones.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
