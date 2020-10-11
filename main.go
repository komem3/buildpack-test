package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
)

type LevelHook struct{}

var _ zerolog.Hook = (*LevelHook)(nil)

// Run is implemented Hook
func (l LevelHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	var levelName string

	switch level {
	case zerolog.DebugLevel:
		levelName = "Debug"
	case zerolog.WarnLevel:
		levelName = "Warning"
	case zerolog.ErrorLevel:
		levelName = "Error"
	default:
		levelName = "Info"
	}
	e.Str("severity", levelName)
}

func main() {
	zerolog.TimestampFieldName = "timeStamp"

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger().Hook(LevelHook{})
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace := r.Header.Get("X-Cloud-Trace-Context")
			splits := strings.Split(trace, "/")
			l := logger.With().Logger()
			if len(splits) >= 2 {
				spans := strings.Split(splits[1], ";")
				if len(spans) >= 2 {
					l = l.With().Str("span", spans[0]).Bool("traceSampled", spans[1] == "o=1").Logger()
				}
				l = l.With().Str("trace", "projects/buld-pack-test/traces/"+splits[0]).Logger()
			}

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
