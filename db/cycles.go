package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const cycleCollection = "cycles"

type Cycle struct {
	Name         string `firestore:"name"`
	OriginalURL  string `firestore:"original_url"`
	ProcessedURL string `firestore:"processed_url"`
}

type Cycles struct {
	Client *firestore.Client
}

func (c *Cycles) Add(ctx context.Context, cycle *Cycle) error {
	if _, _, err := c.Client.Collection(cycleCollection).Add(ctx, cycle); err != nil {
		return fmt.Errorf("could not add cycle: %v", err)
	}
	return nil
}

func (c *Cycles) List(ctx context.Context) ([]*Cycle, error) {
	var cycles []*Cycle
	iter := c.Client.Collection(cycleCollection).Limit(10).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not list cycles: %v", err)
		}
		var c Cycle
		if err := doc.DataTo(&c); err != nil {
			return nil, fmt.Errorf("could not convert doc to cycle: %v", err)
		}
		cycles = append(cycles, &c)
	}
	return cycles, nil
}
