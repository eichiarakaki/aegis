package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
	"github.com/nats-io/nats.go"
)

// RestartSession restarts a FINISHED session.
//
// Component processes are expected to still be alive and connected - they
// receive REBORN (which clears their internal state) and then the orchestrator
// is re-created with the new TimeRange. No new TCP connections or CONFIGURE
// messages are needed.
func RestartSession(session *core.Session, cmd core.Command, conn net.Conn, nc *nats.Conn, tr TimeRange) error {
	if session.GetState() != core.SessionFinished {
		return fmt.Errorf("restart is only valid for FINISHED sessions (current state: %s)", session.GetState())
	}

	// Stop the old orchestrator and data stream.
	if rt, ok := getSessionRuntime(session.ID); ok {
		if rt.orchestrator != nil {
			rt.orchestrator.Stop()
		}
		if rt.dataStream != nil {
			rt.dataStream.Stop()
		}
		clearSessionRuntime(session.ID)
	}

	// Reset session state so SetToRunning is valid later.
	if err := session.ResetToInitialized(); err != nil {
		return fmt.Errorf("restart: reset state: %w", err)
	}

	// Send REBORN to all live components so they clear their domain state.
	// They ACK and then sit idle - waiting for data from the new orchestrator.
	if err := rebornComponents(session, conn); err != nil {
		logger.Warnf("Session %s: reborn had errors: %v", session.ID, err)
	}

	logger.Infof("Session %s: restarting orchestrator (from=%d to=%d)", session.ID, tr.From, tr.To)

	if err := session.SetToStarting(); err != nil {
		return fmt.Errorf("restart: set starting: %w", err)
	}

	if session.Topics == nil || len(*session.Topics) == 0 {
		logger.Warn("Session has no topics — orchestrator will not start")
		return session.SetToRunning()
	}

	topics := *session.Topics

	// Validate that every topic is supported by the session mode before
	// starting any goroutines. Surfaces a clear error to the user instead of
	// silently dropping streams (e.g. "orderBook" in historical mode).
	if err := orchestrator.ValidateTopicsForMode(topics, session.Mode); err != nil {
		session.ForceState(core.SessionInitialized)
		return fmt.Errorf("restart: topic validation: %w", err)
	}

	// Fresh data stream server for the new run.
	ds := orchestrator.NewDataStreamServer(session, nc)
	if err := ds.Start(context.Background()); err != nil {
		session.ForceState(core.SessionInitialized)
		return fmt.Errorf("restart: data stream server: %w", err)
	}

	o, err := orchestrator.New(orchestrator.Config{
		SessionID: session.ID,
		Topics:    topics,
		NC:        nc,
		DS:        ds,
		Mode:      session.Mode,
		Market:    orchestrator.Market(session.Market),
		FromTS:    tr.From,
		ToTS:      tr.To,
	})
	if err != nil {
		ds.Stop()
		session.ForceState(core.SessionInitialized)
		return fmt.Errorf("restart: orchestrator: %w", err)
	}

	o.OnFinished = func() {
		logger.Infof("Session %s: all data exhausted — transitioning to finished", session.ID)
		if err := session.SetToStopping(); err != nil {
			logger.Errorf("Session %s: SetToStopping: %s", session.ID, err)
			return
		}
		if err := session.SetToStopped(); err != nil {
			logger.Errorf("Session %s: SetToStopped: %s", session.ID, err)
			return
		}
		if err := session.SetToFinished(); err != nil {
			logger.Errorf("Session %s: SetToFinished: %s", session.ID, err)
		}
	}
	o.OnError = func(err error) {
		logger.Errorf("Session %s: orchestrator fatal error: %s", session.ID, err)
		_ = session.SetToError()
	}

	setSessionRuntime(session.ID, &sessionRuntime{
		orchestrator: o,
		dataStream:   ds,
	})

	if err := o.Start(context.Background()); err != nil {
		ds.Stop()
		session.ForceState(core.SessionInitialized)
		return fmt.Errorf("restart: start orchestrator: %w", err)
	}

	return session.SetToRunning()
}

// rebornComponents sends REBORN to every registered component and waits for their ACK.
func rebornComponents(session *core.Session, conn net.Conn) error {
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	for _, comp := range session.Registry.List() {
		req := core.NewEnvelope(
			core.MessageTypeLifecycle,
			core.CommandReborn,
			"aegis",
			"component:"+comp.ID,
			map[string]interface{}{},
		)
		if err := enc.Encode(req); err != nil {
			logger.Errorf("reborn: failed to send REBORN to %s (%s): %v", comp.Name, comp.ID, err)
			_ = session.Registry.Unregister(comp.ID)
			continue
		}
		logger.Debugf("reborn: REBORN sent to %s (%s)", comp.Name, comp.ID)

		if err := waitForRebornACK(conn, dec); err != nil {
			logger.Errorf("reborn: no ACK from %s (%s): %v", comp.Name, comp.ID, err)
			continue
		}
		logger.Infof("reborn: %s (%s) ready for new run", comp.Name, comp.ID)
	}
	return nil
}

func waitForRebornACK(conn net.Conn, dec *json.Decoder) error {
	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	defer conn.SetReadDeadline(time.Time{})

	var env core.Envelope
	if err := dec.Decode(&env); err != nil {
		return fmt.Errorf("failed to read ACK: %w", err)
	}
	if err := env.Validate(); err != nil {
		return fmt.Errorf("invalid ACK envelope: %w", err)
	}
	if env.Command != core.CommandACK {
		return fmt.Errorf("expected ACK, got command=%s", env.Command)
	}
	return nil
}
