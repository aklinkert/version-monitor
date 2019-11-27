package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/fielmann-ag/version-monitor/pkg/adapters"
	config2 "github.com/fielmann-ag/version-monitor/pkg/config"
	"github.com/fielmann-ag/version-monitor/pkg/html"
	"github.com/fielmann-ag/version-monitor/pkg/monitor"
	"github.com/fielmann-ag/version-monitor/pkg/version"
)

var (
	config envConfig
	logger *logrus.Logger
)

type envConfig struct {
	Config  string `default:"version-monitor.yaml"`
	Listen  string `default:":8080"`
	Verbose bool
}

func init() {
	envconfig.MustProcess("", &config)

	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	if config.Verbose {
		l.SetLevel(logrus.DebugLevel)
	}
	logger = l
}

func main() {
	if err := adapters.Register(logger); err != nil {
		logger.Fatal(err)
	}

	cfg, err := config2.Load(config.Config)
	if err != nil {
		logger.Fatal(err)
	}

	mon := monitor.NewPeriodic(logger.WithField("section", "monitor"), cfg)
	if err := mon.Start(); err != nil {
		logger.Fatal(err)
	}

	logger.Printf("Listening on %s", config.Listen)
	if err := http.ListenAndServe(config.Listen, router(mon, logger.WithField("section", "http"))); err != nil {
		logger.Fatal(err)
	}
}

func router(mon version.Monitor, logger *logrus.Entry) http.Handler {
	r := mux.NewRouter()
	r.Handle("/", html.NewPageRenderer(mon, logger))
	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/_healthz", func(rw http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(rw, "ok"); err != nil {
			logger.Error(err)
		}
	})

	return r
}
