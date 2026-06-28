package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
	"github.com/elastic/go-elasticsearch/v9"
)

type productSearchDocument struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"`
}

type productSearchRepositoryElasticsearch struct {
	client *elasticsearch.Client
	index  string
}

func NewProductSearchRepositoryElasticsearch(
	client *elasticsearch.Client,
	options *database.ElasticsearchOptions,
) (port.ProductSearchRepository, error) {
	repository := &productSearchRepositoryElasticsearch{
		client: client,
		index:  options.ProductsIndex,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repository.ensureIndex(ctx); err != nil {
		return nil, err
	}

	return repository, nil
}

func (r *productSearchRepositoryElasticsearch) Index(
	ctx context.Context,
	product domain.Product,
) error {
	document := productSearchDocument{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Status:      product.Status,
	}

	payload, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshal product search document: %w", err)
	}

	response, err := r.client.Index(
		r.index,
		bytes.NewReader(payload),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithDocumentID(product.ID),
		r.client.Index.WithRefresh("wait_for"),
	)
	if err != nil {
		return fmt.Errorf("index product search document: %w", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		return elasticsearchResponseError("index product search document", response.Status(), response.Body)
	}

	return nil
}

func (r *productSearchRepositoryElasticsearch) Search(
	ctx context.Context,
	text string,
) ([]domain.Product, error) {
	query := map[string]any{
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  text,
				"fields": []string{"name^2", "description"},
			},
		},
	}

	payload, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal product search query: %w", err)
	}

	response, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.index),
		r.client.Search.WithBody(bytes.NewReader(payload)),
		r.client.Search.WithSize(50),
	)
	if err != nil {
		return nil, fmt.Errorf("search product documents: %w", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		return nil, elasticsearchResponseError("search product documents", response.Status(), response.Body)
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source productSearchDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode product search response: %w", err)
	}

	products := make([]domain.Product, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		products = append(products, domain.Product{
			ID:          hit.Source.ID,
			Name:        hit.Source.Name,
			Description: hit.Source.Description,
			Price:       hit.Source.Price,
			Status:      hit.Source.Status,
		})
	}

	return products, nil
}

func (r *productSearchRepositoryElasticsearch) ensureIndex(ctx context.Context) error {
	response, err := r.client.Indices.Exists(
		[]string{r.index},
		r.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("check product search index: %w", err)
	}
	response.Body.Close()

	if response.StatusCode == 200 {
		return nil
	}
	if response.StatusCode != 404 {
		return fmt.Errorf("check product search index: %s", response.Status())
	}

	mapping := `{
		"mappings": {
			"properties": {
				"id": {"type": "keyword"},
				"name": {"type": "text"},
				"description": {"type": "text"},
				"price": {"type": "double"},
				"status": {"type": "keyword"}
			}
		}
	}`

	response, err = r.client.Indices.Create(
		r.index,
		r.client.Indices.Create.WithContext(ctx),
		r.client.Indices.Create.WithBody(bytes.NewBufferString(mapping)),
	)
	if err != nil {
		return fmt.Errorf("create product search index: %w", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		return elasticsearchResponseError("create product search index", response.Status(), response.Body)
	}

	return nil
}

func elasticsearchResponseError(operation, status string, body io.Reader) error {
	payload, err := io.ReadAll(io.LimitReader(body, 8*1024))
	if err != nil {
		return fmt.Errorf("%s: %s", operation, status)
	}

	return fmt.Errorf("%s: %s: %s", operation, status, string(payload))
}
