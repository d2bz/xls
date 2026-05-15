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

func (c *Client) InsertTag(ctx context.Context, tagNames []string, embeddings [][]float32) error {
	if len(tagNames) == 0 || len(embeddings) == 0 {
		return nil
	}

	dim := int64(len(embeddings[0]))
	if err := c.ensureCollection(ctx, dim); err != nil {
		return fmt.Errorf("ensure collection failed: %w", err)
	}

	_, err := c.cli.Insert(ctx, milvusclient.NewColumnBasedInsertOption(c.collectionName).
		WithVarcharColumn("tag_name", tagNames).
		WithFloatVectorColumn("embedding", int(dim), embeddings))
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	return nil
}

func (c *Client) ensureCollection(ctx context.Context, dim int64) error {
	has, err := c.cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(c.collectionName))
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	schema := entity.NewSchema().
		WithName(c.collectionName).
		WithDynamicFieldEnabled(true).
		WithField(entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeInt64).
			WithIsPrimaryKey(true).
			WithIsAutoID(true)).
		WithField(entity.NewField().
			WithName("tag_name").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(256)).
		WithField(entity.NewField().
			WithName("embedding").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(dim))

	err = c.cli.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(c.collectionName, schema))
	if err != nil {
		return fmt.Errorf("create collection failed: %w", err)
	}

	return nil
}
