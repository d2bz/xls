package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

type Client struct {
	cli            *milvusclient.Client
	collectionName string
}

func NewClient(ctx context.Context, addr, collectionName string) (*Client, error) {
	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus: %w", err)
	}
	return &Client{
		cli:            cli,
		collectionName: collectionName,
	}, nil
}

func (c *Client) Close(ctx context.Context) {
	c.cli.Close(ctx)
}

func (c *Client) SearchTags(ctx context.Context, queryEmbedding []float32, topK int) ([]string, error) {
	if len(queryEmbedding) == 0 {
		return nil, nil
	}
	if topK <= 0 {
		topK = 10
	}

	results, err := c.cli.Search(ctx, milvusclient.NewSearchOption(
		c.collectionName,
		topK,
		[]entity.Vector{entity.FloatVector(queryEmbedding)},
	).WithOutputFields("tag_name").WithANNSField("embedding").WithSearchParam("nprobe", "10"))
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	tagCol := results[0].GetColumn("tag_name")
	if tagCol == nil {
		return nil, nil
	}

	tags := make([]string, 0, results[0].ResultCount)
	for i := 0; i < results[0].ResultCount; i++ {
		tag, err := tagCol.GetAsString(i)
		if err != nil || tag == "" {
			continue
		}
		tags = append(tags, tag)
	}
	return tags, nil
}
