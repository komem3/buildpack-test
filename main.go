package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "survey"
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := logger.With().Interface("header", r.Header).Logger()
			r = r.WithContext(l.WithContext(r.Context()))
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		logger := zerolog.Ctx(r.Context())
		logger.Debug().Interface("params", params).Msg("request param")
		err := json.NewEncoder(w).Encode(params)
		logger.Error().Err(err).Msg("marshal error")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Debug().Str("port", port).Msg("bind port")

	panic(http.ListenAndServe(":"+port, r))
}
