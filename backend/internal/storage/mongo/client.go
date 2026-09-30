// Package mongo implements storage on MongoDB.
package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// serverSelectionTimeout bounds how long an operation waits for a reachable server,
// so a stopped database fails fast instead of hanging requests.
const serverSelectionTimeout = 2 * time.Second

// Client wraps the MongoDB driver client.
type Client struct {
	client *mongo.Client
}

// Connect creates a client for uri. It does not contact the server, so the backend
// starts even while the database is down; only an invalid URI returns an error.
func Connect(uri string) (*Client, error) {
	opts := options.Client().ApplyURI(uri).SetServerSelectionTimeout(serverSelectionTimeout)
	c, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	return &Client{client: c}, nil
}

// Database returns a handle to the named database.
func (c *Client) Database(name string) *mongo.Database {
	return c.client.Database(name)
}

// Ping checks that the primary answers before ctx expires.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongo ping: %w", err)
	}
	return nil
}

// Disconnect closes every connection in the pool.
func (c *Client) Disconnect(ctx context.Context) error {
	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongo disconnect: %w", err)
	}
	return nil
}
