package database

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v9"
)

func NewElasticsearchClient(options *ElasticsearchOptions) (*elasticsearch.Client, error) {
	client, err := elasticsearch.New(elasticsearch.WithAddresses(options.URL))
	if err != nil {
		return nil, fmt.Errorf("create Elasticsearch client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.ConnectionTimeout)
	defer cancel()

	response, err := client.Info(client.Info.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("connect to Elasticsearch: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.IsError() {
		return nil, fmt.Errorf("connect to Elasticsearch: %s", response.Status())
	}

	return client, nil
}
