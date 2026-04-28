package lifecycle

import (
	"errors"
	"fmt"
)

var ErrInvalidTransition = errors.New("invalid status transition")

var serverTransitions = map[string]map[string]bool{
	"": {
		"pending": true,
	},
	"pending": {
		"connected":         true,
		"connection_failed": true,
	},
	"connection_failed": {
		"connected":         true,
		"connection_failed": true,
	},
	"connected": {
		"connected":         true,
		"connection_failed": true,
		"provisioning":      true,
	},
	"provisioning": {
		"ready_for_deploy": true,
		"provision_failed": true,
	},
	"provision_failed": {
		"provisioning":      true,
		"ready_for_deploy":  true,
		"connection_failed": true,
	},
	"ready_for_deploy": {
		"ready_for_deploy": true,
		"provisioning":     true,
	},
}

var siteTransitions = map[string]map[string]bool{
	"": {
		"draft": true,
	},
	"draft": {
		"draft":     true,
		"deploying": true,
	},
	"deploying": {
		"deployed":    true,
		"deploy_failed": true,
		"ssl_pending": true,
	},
	"deploy_failed": {
		"deploying":   true,
		"deploy_failed": true,
	},
	"ssl_pending": {
		"ssl_pending": true,
		"deployed":    true,
	},
	"deployed": {
		"deployed":    true,
		"deploying":   true,
		"ssl_pending": true,
	},
}

func EnsureServerTransition(from, to string) error {
	return ensureTransition(serverTransitions, "server", from, to)
}

func EnsureSiteTransition(from, to string) error {
	return ensureTransition(siteTransitions, "site", from, to)
}

func ensureTransition(transitions map[string]map[string]bool, kind, from, to string) error {
	if to == "" {
		return fmt.Errorf("%w: %s target status is empty", ErrInvalidTransition, kind)
	}
	if from == to {
		return nil
	}

	next, ok := transitions[from]
	if !ok || !next[to] {
		return fmt.Errorf("%w: %s %q -> %q", ErrInvalidTransition, kind, from, to)
	}

	return nil
}
