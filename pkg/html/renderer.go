package html

import (
	"fmt"
	"net/http"

	"github.com/fielmann-ag/ops-version-monitor/pkg/internal/logging"
	"github.com/fielmann-ag/ops-version-monitor/pkg/version"
)

// PageRenderer renders the fetched versions as a simple html page
type PageRenderer struct {
	monitor version.Monitor
	logger  logging.Logger
}

// NewPageRenderer returns a new PageRenderer
func NewPageRenderer(monitor version.Monitor, logger logging.Logger) *PageRenderer {
	return &PageRenderer{
		monitor: monitor,
		logger:  logger,
	}
}

func (r *PageRenderer) render(rw http.ResponseWriter) error {
	versions, date, err := r.monitor.Versions()
	if err != nil {
		return fmt.Errorf("failed to fetch versions from monitor: %v", err)
	}

	params := &pageParams{
		Versions: versions,
		Date: date,
	}
	if err := page.Execute(rw, params); err != nil {
		return fmt.Errorf("failed to render page template: %v", err)
	}

	return nil
}

// ServeHTTP implements the http.Handler interface
func (r *PageRenderer) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	if err := r.render(rw); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		r.logger.Errorf("failed to render versions template: %v", err)
		if _, errWrite := fmt.Fprintf(rw, "failed to render versions template: %v", err); errWrite != nil {
			r.logger.Errorf("error writing error message to response: %v", errWrite)
		}
	}
}
