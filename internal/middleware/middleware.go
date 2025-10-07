package middleware

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

// Логируем HTTP запросы
func LoggingMiddleware(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем wrapper для ResponseWriter чтобы перехватить статус код
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Выполняем следующий обработчик
			next.ServeHTTP(wrapped, r)

			// Логируем запрос
			duration := time.Since(start)
			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"url":         r.URL.String(),
				"status":      wrapped.statusCode,
				"duration":    duration.String(),
				"user_agent":  r.UserAgent(),
				"remote_addr": r.RemoteAddr,
			}).Info("HTTP request")
		})
	}
}

// Для восстанавления от паник
func RecoveryMiddleware(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithFields(logrus.Fields{
						"error":  err,
						"url":    r.URL.String(),
						"method": r.Method,
					}).Error("Panic recovered")

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware() mux.MiddlewareFunc {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // В продакшене указать конкретные домены
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		MaxAge:         86400,
	})

	return c.Handler
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
