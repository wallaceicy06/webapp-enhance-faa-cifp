package db

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/iterator"
)

func TestAddListCycle(t *testing.T) {
	for _, tt := range []struct {
		name    string
		cycle   *Cycle
		want    []*Cycle
		wantErr bool
	}{
		{
			name: "Good",
			cycle: &Cycle{
				Name:         "200326",
				OriginalURL:  "someurl",
				ProcessedURL: "someurl",
			},
			want: []*Cycle{{
				Name:         "200326",
				OriginalURL:  "someurl",
				ProcessedURL: "someurl",
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			testClient := newFirestoreTestClient(context.Background())

			defer func() {
				iter := testClient.Collection(cycleCollection).Documents(ctx)
				batch := testClient.Batch()
				for {
					doc, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						t.Fatalf("could not read document: %v", err)
					}

					batch.Delete(doc.Ref)
				}
				if _, err := batch.Commit(ctx); err != nil {
					t.Fatalf("could not commit batch: %v", err)
				}
			}()

			cyclesDb := &Cycles{
				Client: testClient,
			}

			if err := cyclesDb.Add(ctx, tt.cycle); err != nil {
				t.Errorf("could not add entity: %v", err)
			}
			got, err := cyclesDb.List(ctx)
			if err != nil {
				t.Errorf("could not list cycles: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("cycles diff (-got +want): %s", diff)
			}
		})
	}
}

func newFirestoreTestClient(ctx context.Context) *firestore.Client {
	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		log.Fatalf("firebase.NewClient err: %v", err)
	}

	return client
}

func TestMain(m *testing.M) {
	const FirestoreEmulatorHost = "FIRESTORE_EMULATOR_HOST"

	cmd := exec.Command("gcloud", "beta", "emulators", "firestore", "start", "--host-port=localhost")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		buf := make([]byte, 256, 256)
		for {
			n, err := stderr.Read(buf[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("reading stderr %v", err)
			}

			if n > 0 {
				d := string(buf[:n])

				log.Printf("%s", d)

				if strings.Contains(d, "Dev App Server is now running") {
					wg.Done()
				}

				pos := strings.Index(d, FirestoreEmulatorHost+"=")
				if pos > 0 {
					host := d[pos+len(FirestoreEmulatorHost)+1 : len(d)-1]
					os.Setenv(FirestoreEmulatorHost, host)
				}
			}
		}
	}()

	wg.Wait()

	result := m.Run()

	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	os.Exit(result)
}
