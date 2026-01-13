// Package qdrant provides Qdrant vector database integration activities for resolute workflows.
package qdrant

import (
	"github.com/resolute-sh/resolute/core"
	"go.temporal.io/sdk/worker"
)

const (
	ProviderName    = "resolute-qdrant"
	ProviderVersion = "1.0.0"
)

// Provider returns the Qdrant provider for registration.
func Provider() core.Provider {
	return core.NewProvider(ProviderName, ProviderVersion).
		AddActivity("qdrant.Upsert", UpsertActivity).
		AddActivity("qdrant.Search", SearchActivity).
		AddActivity("qdrant.Delete", DeleteActivity).
		AddActivity("qdrant.CreateCollection", CreateCollectionActivity)
}

// RegisterActivities registers all Qdrant activities with a Temporal worker.
func RegisterActivities(w worker.Worker) {
	core.RegisterProviderActivities(w, Provider())
}
