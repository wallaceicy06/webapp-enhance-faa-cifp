package blob

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// GCSClient is a client for writing new objects to a GCS bucket.
type GCSClient struct {
	Client     *storage.Client
	BucketName string
}

// NewObject creates a new object with the specified file name in the bucket for this
// GCS client. If public is true, then the file is set to be readable by all users.
func (g *GCSClient) NewObject(ctx context.Context, fileName string, public bool) (io.WriteCloser, error) {
	obj := g.Client.Bucket(g.BucketName).Object(fileName)
	if public {
		if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			return nil, fmt.Errorf("could not set ACL for new object %q", fileName)
		}
	}
	return obj.NewWriter(ctx), nil
}
