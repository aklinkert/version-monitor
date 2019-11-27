package version

import (
	"fmt"
	"sync"
	"time"

	"github.com/fielmann-ag/ops-version-monitor/pkg/config"
	"github.com/fielmann-ag/ops-version-monitor/pkg/internal/logging"

	"github.com/robfig/cron/v3"
)

// PeriodicMonitor periodically iterates the map of adapters and updates the fetched versions
type PeriodicMonitor struct {
	sync.RWMutex
	logger           logging.Logger
	config           *config.Config
	cachedVersions   map[string]Version
	lastError        error
	latestResultFrom time.Time
}

// NewPeriodicMonitor returns a new fetcher instance
func NewPeriodicMonitor(logger logging.Logger, config *config.Config) *PeriodicMonitor {
	return &PeriodicMonitor{
		logger:         logger,
		config:         config,
		cachedVersions: map[string]Version{},
	}
}

// Versions returns the latest set of versions cached since the last update
func (m *PeriodicMonitor) Versions() ([]Version, time.Time, error) {
	m.RLock()
	if m.lastError != nil {
		m.RUnlock()
		return nil, time.Time{}, m.lastError
	}

	versions := make([]Version, 0)
	for _, version := range m.cachedVersions {
		versions = append(versions, version)
	}

	m.RUnlock()
	return versions, m.latestResultFrom, nil
}

func (m *PeriodicMonitor) validateConfig() error {
	for _, t := range m.config.Targets {
		if _, ok := adapters[t.Latest.Type]; !ok {
			return fmt.Errorf("target.latest.type %s of target %s not found", t.Latest.Type, t.Name)
		}
		if _, ok := adapters[t.Current.Type]; !ok {
			return fmt.Errorf("target.current.type %s of target %s not found", t.Current.Type, t.Name)
		}
	}

	return nil
}

// Start the periodic fetching
func (m *PeriodicMonitor) Start() error {
	if err := m.validateConfig(); err != nil {
		return err
	}

	c := cron.New()
	if _, err := c.AddJob("@hourly", m); err != nil {
		return fmt.Errorf("failed to register cron job: %v", err)
	}

	c.Start()
	m.Run()
	return nil
}

// Run fetch from all adapters
func (m *PeriodicMonitor) Run() {
	m.logger.Debugf("start fetching versions ...")

	for _, target := range m.config.Targets {
		go m.fetch(target)
	}

	m.logger.Debugf("done fetching versions.")
}

func (m *PeriodicMonitor) fetch(target config.Target) {
	m.logger.Debugf("fetching version %v", target.Name)

	currentVersionAdapter := adapters[target.Current.Type]
	currentVersion, err := currentVersionAdapter.Fetch(target.Current)
	if err != nil {
		m.error(fmt.Errorf("failed to load version from target.Current adapter %v: %v", target.Current.Type, err))
		return
	}

	latestVersionAdapter := adapters[target.Latest.Type]
	latestVersion, err := latestVersionAdapter.Fetch(target.Latest)
	if err != nil {
		m.error(fmt.Errorf("failed to load version from target.Latest adapter %v: %v", target.Latest.Type, err))
		return
	}

	m.storeVersion(target.Name, Version{
		Name:    target.Name,
		Current: currentVersion,
		Latest:  latestVersion,
	})
	m.logger.Debugf("fetching version %v done", target.Name)
}

func (m *PeriodicMonitor) storeVersion(name string, version Version) {
	m.Lock()
	m.cachedVersions[name] = version
	m.Unlock()
}

func (m *PeriodicMonitor) error(err error) {
	m.logger.Error(err)
	m.Lock()
	m.lastError = err
	m.Unlock()
}
