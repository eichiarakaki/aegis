package health

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/nats-io/nats.go"
)

var daemonStartedAt = time.Now()

// GlobalHealth builds the daemon-level health report.
func GlobalHealth(sessionStore *core.SessionStore, nc *nats.Conn) core.HealthGlobalData {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	sessions := sessionStore.ListSessions()
	sh := core.SessionHealthSummary{Total: len(sessions)}
	ch := core.ComponentHealthSummary{}

	for _, s := range sessions {
		switch s.GetState() {
		case core.SessionRunning:
			sh.Running++
		case core.SessionInitialized:
			sh.Initialized++
		case core.SessionStopped, core.SessionFinished:
			sh.Stopped++
		case core.SessionError:
			sh.Error++
		}
		if s.Registry == nil {
			continue
		}
		for _, c := range s.Registry.List() {
			ch.Total++
			switch c.State {
			case core.ComponentStateRunning:
				ch.Running++
			case core.ComponentStateError:
				ch.Error++
			case core.ComponentStateInit:
				ch.Init++
			}
		}
	}

	natsHealth := core.NATSHealth{Connected: false}
	if nc != nil {
		natsHealth.Connected = nc.IsConnected()
		natsHealth.URL = nc.ConnectedUrl()
	}

	status := "healthy"
	if !natsHealth.Connected || sh.Error > 0 || ch.Error > 0 {
		status = "degraded"
	}

	return core.HealthGlobalData{
		Status:        status,
		UptimeSeconds: int64(time.Since(daemonStartedAt).Seconds()),
		Daemon: core.DaemonHealth{
			PID:       os.Getpid(),
			MemoryRSS: memStats.Sys,
		},
		NATS:       natsHealth,
		Sessions:   sh,
		Components: ch,
	}
}

// SessionHealth builds the session-level health report.
func SessionHealth(
	session *core.Session,
	pool *servicescomponent.ConnectionPool,
	heartbeatTimeout time.Duration,
) core.HealthSessionData {
	compRefs := make([]core.ComponentHealthRef, 0)
	anyError := false

	for _, c := range session.Registry.List() {
		_, connActive := pool.Get(c.ID)
		secsSince := time.Since(c.LastHeartbeat).Seconds()
		hbOK := c.State == core.ComponentStateInit ||
			c.State == core.ComponentStateRegistered ||
			c.State == core.ComponentStateInitializing ||
			secsSince < heartbeatTimeout.Seconds()

		var uptime int64
		if !c.StartedAt.IsZero() {
			uptime = int64(time.Since(c.StartedAt).Seconds())
		}

		if c.State == core.ComponentStateError || !hbOK {
			anyError = true
		}

		compRefs = append(compRefs, core.ComponentHealthRef{
			ID:                 c.ID,
			Name:               c.Name,
			State:              string(c.State),
			UptimeSeconds:      uptime,
			SecsSinceHeartbeat: secsSince,
			HeartbeatOK:        hbOK,
			ConnectionActive:   connActive,
		})
	}

	// Data stream socket
	socketPath := session.GetStreamSocketPath()
	socketExists := false
	if socketPath != "" {
		_, err := os.Stat(socketPath)
		socketExists = err == nil
	}

	topicCount := 0
	if session.Topics != nil {
		topicCount = len(*session.Topics)
	}

	dsHealth := core.DataStreamHealth{
		SocketPath:   socketPath,
		SocketExists: socketExists,
		TopicCount:   topicCount,
	}

	// Data files — only relevant for historical mode
	var dfHealth *core.DataFilesHealth
	if session.Mode == "historical" {
		dfHealth = checkDataFiles(session)
		if dfHealth.FilesMissing > 0 {
			anyError = true
		}
	}

	sessionState := session.GetState()
	status := "healthy"
	switch {
	case sessionState == core.SessionInitialized || sessionState == core.SessionStopped:
		status = "inactive"
	case anyError:
		status = "degraded"
	}

	return core.HealthSessionData{
		Status: status,
		Session: core.SessionDetail{
			ID:            session.ID,
			Name:          session.Name,
			Mode:          session.Mode,
			State:         string(sessionState),
			UptimeSeconds: session.GetUptimeSeconds(),
			CreatedAt:     session.CreatedAt,
			StartedAt:     session.StartedAt,
			StoppedAt:     session.StoppedAt,
		},
		Components: compRefs,
		DataStream: dsHealth,
		DataFiles:  dfHealth,
	}
}

// ComponentHealth builds the component-level health report.
func ComponentHealth(
	session *core.Session,
	comp *core.Component,
	pool *servicescomponent.ConnectionPool,
	heartbeatTimeout time.Duration,
) core.HealthComponentData {
	_, connActive := pool.Get(comp.ID)
	secsSince := time.Since(comp.LastHeartbeat).Seconds()
	hbOK := comp.State == core.ComponentStateInit ||
		comp.State == core.ComponentStateRegistered ||
		comp.State == core.ComponentStateInitializing ||
		secsSince < heartbeatTimeout.Seconds()

	var uptime int64
	if !comp.StartedAt.IsZero() {
		uptime = int64(time.Since(comp.StartedAt).Seconds())
	}

	status := "healthy"
	switch {
	case comp.State == core.ComponentStateInit ||
		comp.State == core.ComponentStateRegistered:
		status = "inactive"
	case comp.State == core.ComponentStateError || !hbOK:
		status = "degraded"
	}

	return core.HealthComponentData{
		Status:             status,
		SessionID:          session.ID,
		ID:                 comp.ID,
		Name:               comp.Name,
		State:              string(comp.State),
		UptimeSeconds:      uptime,
		LastHeartbeat:      comp.LastHeartbeat,
		SecsSinceHeartbeat: secsSince,
		HeartbeatOK:        hbOK,
		ConnectionActive:   connActive,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// checkDataFiles verifies that CSV files exist for every configured
// currency+timeframe combination under config.DataPath.
func checkDataFiles(session *core.Session) *core.DataFilesHealth {
	cfg, err := config.LoadAegis()
	if err != nil {
		return &core.DataFilesHealth{DataPath: "unavailable"}
	}

	found := 0
	missing := 0
	var missingList []string

	for _, currency := range cfg.Fetcher.Cryptocurrencies {
		for _, tf := range currency.Intervals {
			// Expected pattern: <DataPath>/<SYMBOL>/<SYMBOL>_<timeframe>.csv
			pattern := filepath.Join(cfg.DataPath, strings.ToUpper(currency.Symbol),
				fmt.Sprintf("%s_%s.csv", strings.ToUpper(currency.Symbol), tf))
			matches, err := filepath.Glob(pattern)
			if err != nil || len(matches) == 0 {
				missing++
				missingList = append(missingList, pattern)
			} else {
				found += len(matches)
			}
		}
	}

	result := &core.DataFilesHealth{
		DataPath:     cfg.DataPath,
		FilesFound:   found,
		FilesMissing: missing,
	}
	if len(missingList) > 0 {
		result.Missing = missingList
	}
	return result
}
