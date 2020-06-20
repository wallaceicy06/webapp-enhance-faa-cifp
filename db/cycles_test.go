package db

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/go-cmp/cmp"
	"github.com/phayes/freeport"
	"google.golang.org/api/iterator"
)

func cleanUp(t *testing.T, ctx context.Context, client *firestore.Client) {
	t.Helper()
	iter := client.Collection(cycleCollection).Documents(ctx)
	batch := client.Batch()
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
}

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
			defer cleanUp(t, ctx, testClient)

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

func TestAddGet(t *testing.T) {
	allCycles := []*Cycle{
		{
			Name:         "200326",
			OriginalURL:  "someurl",
			ProcessedURL: "someurl",
		},
		{
			Name:         "200425",
			OriginalURL:  "someurl",
			ProcessedURL: "someurl",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testClient := newFirestoreTestClient(context.Background())
	defer cleanUp(t, ctx, testClient)

	cyclesDb := &Cycles{
		Client: testClient,
	}

	for _, c := range allCycles {
		if err := cyclesDb.Add(ctx, c); err != nil {
			t.Fatalf("could not add entity: %v", err)
		}
	}
	got, err := cyclesDb.Get(ctx, "200425")
	if err != nil {
		t.Errorf("could not list cycles: %v", err)
	}
	want := &Cycle{
		Name:         "200425",
		OriginalURL:  "someurl",
		ProcessedURL: "someurl",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Get() = %+v, _ want %+v, _", got, want)
	}

	got, err = cyclesDb.Get(ctx, "doesnotexist")
	if err != nil {
		t.Errorf("could not list cycles: %v", err)
	}
	if got != nil {
		t.Errorf("Get() = %+v, _ want <nil>, _", got)
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
	const firestoreEmulatorHost = "FIRESTORE_EMULATOR_HOST"

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatalf("could not find open port: %v", err)
	}

	host := fmt.Sprintf("localhost:%d", port)
	oldHost := os.Getenv(firestoreEmulatorHost)
	defer func() {
		os.Setenv(firestoreEmulatorHost, oldHost)
	}()
	os.Setenv(firestoreEmulatorHost, host)

	cmd := exec.Command("gcloud", "beta", "emulators", "firestore", "start", fmt.Sprintf("--host-port=localhost:%d", port))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

	var ok bool
	for i := 0; i < 10; i++ {
		log.Printf("Attempting to connect to Firestore server %q, attempt %d.", host, i)
		res, err := http.Get("http://" + host)
		if err == nil && res.StatusCode == http.StatusOK {
			ok = true
			break
		}
		time.Sleep(5 * time.Second)
	}
	if !ok {
		log.Fatalf("Problem starting Firestore server, response not OK.")
	}

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
				log.Printf("%s", buf[:n])
			}
		}
	}()

	result := m.Run()

	os.Exit(result)
}
