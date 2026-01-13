package qdrant

import (
	"context"
	"fmt"

	"github.com/resolute-sh/resolute/core"
	transform "github.com/resolute-sh/resolute-transform"
)

// UpsertInput is the input for UpsertActivity.
type UpsertInput struct {
	BaseURL       string
	APIKey        string
	Collection    string
	EmbeddingsRef core.DataRef
}

// PointInput represents a point to upsert.
type PointInput struct {
	ID       string
	Vector   []float64
	Metadata map[string]string
}

// UpsertOutput is the output of UpsertActivity.
type UpsertOutput struct {
	Count   int
	Success bool
}

// EmbeddedDocumentForLoad represents loaded embedding data.
type EmbeddedDocumentForLoad struct {
	Document   transform.Document `json:"Document"`
	Embedding  []float64          `json:"Embedding"`
	Dimensions int                `json:"Dimensions"`
}

// UpsertActivity upserts embeddings from a DataRef into a Qdrant collection.
func UpsertActivity(ctx context.Context, input UpsertInput) (UpsertOutput, error) {
	storage, err := core.GetStorage()
	if err != nil {
		return UpsertOutput{}, fmt.Errorf("get storage: %w", err)
	}

	var embeddings []EmbeddedDocumentForLoad
	if err := storage.LoadJSON(ctx, input.EmbeddingsRef, &embeddings); err != nil {
		return UpsertOutput{}, fmt.Errorf("load embeddings: %w", err)
	}

	client := NewClient(ClientConfig{
		BaseURL: input.BaseURL,
		APIKey:  input.APIKey,
	})

	points := make([]Point, 0, len(embeddings))
	for _, emb := range embeddings {
		points = append(points, Point{
			ID:      emb.Document.ID,
			Vector:  emb.Embedding,
			Payload: emb.Document.Metadata,
		})
	}

	if err := client.Upsert(ctx, input.Collection, points); err != nil {
		return UpsertOutput{}, fmt.Errorf("upsert points: %w", err)
	}

	return UpsertOutput{
		Count:   len(points),
		Success: true,
	}, nil
}

// UpsertPointsInput is the input for UpsertPointsActivity (direct points).
type UpsertPointsInput struct {
	BaseURL    string
	APIKey     string
	Collection string
	Points     []PointInput
}

// UpsertPointsActivity upserts points directly into a Qdrant collection.
func UpsertPointsActivity(ctx context.Context, input UpsertPointsInput) (UpsertOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL: input.BaseURL,
		APIKey:  input.APIKey,
	})

	points := make([]Point, 0, len(input.Points))
	for _, p := range input.Points {
		points = append(points, Point{
			ID:      p.ID,
			Vector:  p.Vector,
			Payload: p.Metadata,
		})
	}

	if err := client.Upsert(ctx, input.Collection, points); err != nil {
		return UpsertOutput{}, fmt.Errorf("upsert points: %w", err)
	}

	return UpsertOutput{
		Count:   len(points),
		Success: true,
	}, nil
}

// SearchInput is the input for SearchActivity.
type SearchInput struct {
	BaseURL    string
	APIKey     string
	Collection string
	Vector     []float64
	Limit      int
}

// SearchOutput is the output of SearchActivity.
type SearchOutput struct {
	Results []SearchResultOutput
	Count   int
}

// SearchResultOutput represents a search result.
type SearchResultOutput struct {
	ID       string
	Score    float64
	Metadata map[string]string
}

// SearchActivity performs a vector similarity search.
func SearchActivity(ctx context.Context, input SearchInput) (SearchOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL: input.BaseURL,
		APIKey:  input.APIKey,
	})

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	results, err := client.Search(ctx, input.Collection, input.Vector, limit)
	if err != nil {
		return SearchOutput{}, fmt.Errorf("search: %w", err)
	}

	output := make([]SearchResultOutput, 0, len(results))
	for _, r := range results {
		output = append(output, SearchResultOutput{
			ID:       r.ID,
			Score:    r.Score,
			Metadata: r.Payload,
		})
	}

	return SearchOutput{
		Results: output,
		Count:   len(output),
	}, nil
}

// DeleteInput is the input for DeleteActivity.
type DeleteInput struct {
	BaseURL    string
	APIKey     string
	Collection string
	IDs        []string
}

// DeleteOutput is the output of DeleteActivity.
type DeleteOutput struct {
	Deleted int
	Success bool
}

// DeleteActivity deletes points from a Qdrant collection.
func DeleteActivity(ctx context.Context, input DeleteInput) (DeleteOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL: input.BaseURL,
		APIKey:  input.APIKey,
	})

	if err := client.Delete(ctx, input.Collection, input.IDs); err != nil {
		return DeleteOutput{}, fmt.Errorf("delete points: %w", err)
	}

	return DeleteOutput{
		Deleted: len(input.IDs),
		Success: true,
	}, nil
}

// CreateCollectionInput is the input for CreateCollectionActivity.
type CreateCollectionInput struct {
	BaseURL    string
	APIKey     string
	Name       string
	VectorSize int
	Distance   string
}

// CreateCollectionOutput is the output of CreateCollectionActivity.
type CreateCollectionOutput struct {
	Name    string
	Created bool
}

// CreateCollectionActivity creates a new Qdrant collection.
func CreateCollectionActivity(ctx context.Context, input CreateCollectionInput) (CreateCollectionOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL: input.BaseURL,
		APIKey:  input.APIKey,
	})

	if err := client.CreateCollection(ctx, input.Name, input.VectorSize, input.Distance); err != nil {
		return CreateCollectionOutput{}, fmt.Errorf("create collection: %w", err)
	}

	return CreateCollectionOutput{
		Name:    input.Name,
		Created: true,
	}, nil
}

// Upsert creates a node for upserting points.
func Upsert(input UpsertInput) *core.Node[UpsertInput, UpsertOutput] {
	return core.NewNode("qdrant.Upsert", UpsertActivity, input)
}

// Search creates a node for searching.
func Search(input SearchInput) *core.Node[SearchInput, SearchOutput] {
	return core.NewNode("qdrant.Search", SearchActivity, input)
}

// Delete creates a node for deleting points.
func Delete(input DeleteInput) *core.Node[DeleteInput, DeleteOutput] {
	return core.NewNode("qdrant.Delete", DeleteActivity, input)
}

// CreateCollection creates a node for creating a collection.
func CreateCollection(input CreateCollectionInput) *core.Node[CreateCollectionInput, CreateCollectionOutput] {
	return core.NewNode("qdrant.CreateCollection", CreateCollectionActivity, input)
}
