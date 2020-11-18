package blob

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

// GCSClient is a client for writing new objects to a GCS bucket.
type GCSClient struct {
	Client     *storage.Client
	BucketName string
}

// NewObject creates a new object with the specified file name in the bucket for this
// GCS client and returns a writer for the object.
func (g *GCSClient) NewObject(ctx context.Context, fileName string) io.WriteCloser {
	return g.Client.Bucket(g.BucketName).Object(fileName).NewWriter(ctx)
}

// AllowPublicAccess sets the ACL on the specified file to be public. The object must
// already exist.
func (g *GCSClient) AllowPublicAccess(ctx context.Context, fileName string) error {
	return g.Client.Bucket(g.BucketName).Object(fileName).ACL().Set(ctx, storage.AllUsers, storage.RoleReader)
}
