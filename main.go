package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type levelHook struct{}

var _ zerolog.Hook = (*levelHook)(nil)

// Run is implemented Hook
func (l levelHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
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
	zerolog.TimeFieldFormat = time.RFC3339Nano

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger().Hook(levelHook{})
	if !metadata.OnGCE() {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace := r.Header.Get("X-Cloud-Trace-Context")
			splits := strings.Split(trace, "/")
			l := logger.With().Logger()
			if len(splits) >= 2 {
				spans := strings.Split(splits[1], ";")
				if len(spans) >= 2 {
					l = l.With().Str("logging.googleapis.com/spanId", spans[0]).Bool("logging.googleapis.com/trace_sampled", spans[1] == "o=1").Logger()
				}
				l = l.With().Str("logging.googleapis.com/trace", "projects/buld-pack-test/traces/"+splits[0]).Logger()
			}

			r = r.WithContext(l.WithContext(r.Context()))
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		logger := zerolog.Ctx(r.Context())
		logger.Debug().Interface("params", params).Msg("request param")
		_, err := os.Open("nofile")
		err = errors.Wrap(err, "open file")
		logger.Err(err).Msg("openfile")

		if len(params) == 0 {
			w.Write([]byte("no params"))
			return
		}
		if err = json.NewEncoder(w).Encode(params); err != nil {
			logger.Error().Err(err).Msg("marshal error")
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Debug().Str("port", port).Msg("bind port")

	panic(http.ListenAndServe(":"+port, r))
}
