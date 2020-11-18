package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const cycleCollection = "cycles"

type Cycle struct {
	Name      string    `firestore:"name"`
	Original  string    `firestore:"original"`
	Processed string    `firestore:"processed"`
	Date      time.Time `firestore:"date"`
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

func (c *Cycles) Get(ctx context.Context, name string) (*Cycle, error) {
	iter := c.Client.Collection(cycleCollection).Where("name", "==", name).Documents(ctx)
	var cycle Cycle
	var i int
	for i = 0; ; i++ {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if i > 0 {
			log.Printf("More than one item in database with name %q.", name)
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not list cycles: %v", err)
		}
		if err := doc.DataTo(&cycle); err != nil {
			return nil, fmt.Errorf("could not convert doc to cycle: %v", err)
		}
	}
	if i == 0 {
		return nil, nil
	}
	return &cycle, nil
}

func (c *Cycles) List(ctx context.Context) ([]*Cycle, error) {
	var cycles []*Cycle
	iter := c.Client.Collection(cycleCollection).OrderBy("date", firestore.Desc).Limit(10).Documents(ctx)
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
