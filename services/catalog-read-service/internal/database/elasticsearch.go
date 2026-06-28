package database

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v9"
)

func NewElasticsearchClient(options *ElasticsearchOptions) (*elasticsearch.Client, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{options.URL},
	})
	if err != nil {
		return nil, fmt.Errorf("create Elasticsearch client: %w", err)
	}

	response, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("connect to Elasticsearch: %w", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		return nil, fmt.Errorf("connect to Elasticsearch: %s", response.Status())
	}

	return client, nil
}
