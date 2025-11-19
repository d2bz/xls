package es

import (
	"github.com/elastic/go-elasticsearch/v8"
)

type (
	Config struct {
		Address  []string
		Username string
		Password string
	}

	Es struct {
		*elasticsearch.Client
	}

	TypedEs struct {
		*elasticsearch.TypedClient
	}
)

func NewEs(c *Config) (*Es, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: c.Address,
		Username:  c.Username,
		Password:  c.Password,
	})
	if err != nil {
		return nil, err
	}
	return &Es{client}, nil
}

func MustNewEs(c *Config) *Es {
	es, err := NewEs(c)
	if err != nil {
		panic(err)
	}
	return es
}

func NewTypedEs(c *Config) (*TypedEs, error) {
	TypedClient, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: c.Address,
		Username:  c.Username,
		Password:  c.Password,
	})
	if err != nil {
		return nil, err
	}
	return &TypedEs{TypedClient}, nil
}

func MustNewTypedEs(c *Config) *TypedEs {
	typed, err := NewTypedEs(c)
	if err != nil {
		panic(err)
	}
	return typed
}
