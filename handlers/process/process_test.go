package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
)

type fakeVerifier struct {
	GotToken string
	Email    string
	Err      error
}

func (fv *fakeVerifier) VerifyGoogle(_ context.Context, token string) (string, error) {
	fv.GotToken = token
	return fv.Email, fv.Err
}

type fakeCyclesAdderGetter struct {
	AddedCycle *db.Cycle
	AddErr     error
	GetCycle   *db.Cycle
	GetErr     error
}

func (ag *fakeCyclesAdderGetter) Add(_ context.Context, c *db.Cycle) error {
	ag.AddedCycle = c
	return ag.AddErr
}

func (ag *fakeCyclesAdderGetter) Get(context.Context, string) (*db.Cycle, error) {
	return ag.GetCycle, ag.GetErr
}

type nopWriteCloser struct {
	io.Writer
}

func (*nopWriteCloser) Close() error { return nil }

type fakeStorageWriter struct {
	Original  bytes.Buffer
	Processed bytes.Buffer
}

func (fs *fakeStorageWriter) GetStorageWriter(_ context.Context, bucket, objectName string) io.WriteCloser {
	if strings.Contains(objectName, "original") {
		return &nopWriteCloser{&fs.Original}
	} else if strings.Contains(objectName, "processed") {
		return &nopWriteCloser{&fs.Processed}
	}
	return nil
}

type fakeCifpServerConfig struct {
	EditionsRes     func(url string) string
	EditionsErr     bool
	CifpFileData    []byte
	CifpFileDataErr bool
}

type fakeCifpServer struct {
	config  *fakeCifpServerConfig
	baseURL string
}

func (fcs *fakeCifpServer) HandleEditions(w http.ResponseWriter, r *http.Request) {
	if fcs.config.EditionsErr {
		http.Error(w, "Interal error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, fcs.config.EditionsRes(fcs.baseURL+"/upload/cifp/current"))
}

func (fcs *fakeCifpServer) HandleFileData(w http.ResponseWriter, r *http.Request) {
	if fcs.config.CifpFileDataErr {
		http.Error(w, "Interal error", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(fcs.config.CifpFileData); err != nil {
		http.Error(w, "Could not write file data", http.StatusInternalServerError)
		log.Printf("Could not write file data: %v", err)
		return
	}
}

func newFakeCifpServer(config *fakeCifpServerConfig) *httptest.Server {
	srv := &fakeCifpServer{config: config}
	mux := http.NewServeMux()
	mux.HandleFunc("/apra/cifp/chart", srv.HandleEditions)
	mux.HandleFunc("/upload/cifp/current", srv.HandleFileData)
	testServer := httptest.NewServer(mux)
	srv.baseURL = testServer.URL
	return testServer
}

const (
	testDataZipFile           = "original.zip"
	wantTestDataProcessedFile = "want_processed.txt"
)

const goodEditionResTmpl = `{
	  "edition": [{
        "editionName": "CURRENT",
        "format": "ZIP",
        "editionDate": "06/18/2020",
        "editionNumber": 7,
        "product": {
          "productName": "CIFP",
          "url": %q
        }
      }]
    }`

func goodEditionsRes(url string) string { return fmt.Sprintf(goodEditionResTmpl, url) }

func TestHandleAuth(t *testing.T) {
	const serviceAccountEmail = "some-email@example.com"
	cifpZipData, err := ioutil.ReadFile(testDataZipFile)
	if err != nil {
		t.Fatalf("Could not read data file: %v", err)
	}

	for _, tt := range []struct {
		name         string
		authHeader   string
		fakeVerifier *fakeVerifier
		wantStatus   int
	}{
		{
			name:       "Good",
			authHeader: "Bearer token",
			fakeVerifier: &fakeVerifier{
				Email: "some-email@example.com",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "NoAuthorization",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "BadEmail",

			authHeader: "Bearer token",
			fakeVerifier: &fakeVerifier{
				Email: "some-email@evil.com",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "VerificationError",
			authHeader: "Bearer token",
			fakeVerifier: &fakeVerifier{
				Err: errors.New("problem verifying token"),
			},
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeCifpServer(
				&fakeCifpServerConfig{
					EditionsRes:  goodEditionsRes,
					CifpFileData: cifpZipData,
				},
			)
			defer srv.Close()

			rr := httptest.NewRecorder()
			fsw := &fakeStorageWriter{}
			handler := &Handler{
				ServiceAccountEmail: serviceAccountEmail,
				Verifier:            tt.fakeVerifier,
				Cycles:              &fakeCyclesAdderGetter{},
				CifpURL:             srv.URL + "/apra/cifp/chart",
				GetStorageWriter:    fsw.GetStorageWriter,
			}
			req := httptest.NewRequest(http.MethodPost, "/", &bytes.Buffer{})
			req.Header.Set("Authorization", tt.authHeader)
			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %d want %d",
					status, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && tt.fakeVerifier.GotToken != "token" {
				t.Errorf(`verifier received token %q want "token"`, tt.fakeVerifier.GotToken)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	cifpZipData, err := ioutil.ReadFile(testDataZipFile)
	if err != nil {
		t.Fatalf("Could not read data file: %v", err)
	}
	wantProcessedData, err := ioutil.ReadFile(wantTestDataProcessedFile)
	if err != nil {
		t.Fatalf("Could not read data file: %v", err)
	}

	for _, tt := range []struct {
		name                 string
		fakeCifpServerConfig *fakeCifpServerConfig
		fakeCycles           *fakeCyclesAdderGetter
		wantStatus           int
		wantSkipProcess      bool
		wantAddCycle         *db.Cycle
	}{
		{
			name: "Good",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes:  goodEditionsRes,
				CifpFileData: cifpZipData,
			},
			fakeCycles: &fakeCyclesAdderGetter{},
			wantStatus: http.StatusOK,
			wantAddCycle: &db.Cycle{
				Name:         "06/18/2020",
				OriginalURL:  "https://storage.googleapis.com/faa-cifp-data/original/FAACIFP18_original_06-18-2020.zip",
				ProcessedURL: "https://storage.googleapis.google.com/faa-cifp-data/processed/FAACIFP18_processed_06-18-2020",
				Date:         time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "EditionsInvalidJson",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes: func(string) string {
					return `}{`
				},
				CifpFileData: cifpZipData,
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "NoEditions",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes: func(string) string {
					return `{ "edition": [] }`
				},
				CifpFileData: cifpZipData,
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "InvalidZipData",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes:  goodEditionsRes,
				CifpFileData: []byte("bleh"),
			},
			fakeCycles: &fakeCyclesAdderGetter{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "CycleExists",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes:  goodEditionsRes,
				CifpFileData: cifpZipData,
			},
			fakeCycles: &fakeCyclesAdderGetter{
				GetCycle: &db.Cycle{Name: "06/18/2020"},
			},
			wantStatus:      http.StatusOK,
			wantSkipProcess: true,
		},
		{
			name: "CycleGetError",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes:  goodEditionsRes,
				CifpFileData: cifpZipData,
			},
			fakeCycles: &fakeCyclesAdderGetter{
				GetErr: errors.New("problem fetching cycles"),
			},
			wantStatus:      http.StatusInternalServerError,
			wantSkipProcess: true,
		},
		{
			name: "CycleAddError",
			fakeCifpServerConfig: &fakeCifpServerConfig{
				EditionsRes:  goodEditionsRes,
				CifpFileData: cifpZipData,
			},
			fakeCycles: &fakeCyclesAdderGetter{
				AddErr: errors.New("problem adding cycle"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeCifpServer(tt.fakeCifpServerConfig)
			defer srv.Close()

			rr := httptest.NewRecorder()
			fsw := &fakeStorageWriter{}
			handler := &Handler{
				ServiceAccountEmail: "some-email@example.com",
				Verifier: &fakeVerifier{
					Email: "some-email@example.com",
				},
				Cycles:           tt.fakeCycles,
				CifpURL:          srv.URL + "/apra/cifp/chart",
				GetStorageWriter: fsw.GetStorageWriter,
			}
			req := httptest.NewRequest(http.MethodPost, "/", &bytes.Buffer{})
			req.Header.Set("Authorization", "Bearer token")
			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %d want %d",
					status, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			if tt.wantSkipProcess {
				if fsw.Original.Len() != 0 {
					t.Error("wanted processing to be skipped, but got original data")
				}
				if fsw.Processed.Len() != 0 {
					t.Error("wanted processing to be skipped, but got processed data")
				}
				return
			}
			if !bytes.Equal(cifpZipData, fsw.Original.Bytes()) {
				t.Error("original data not the same as input zip data")
			}
			if diff := cmp.Diff(wantProcessedData, fsw.Processed.Bytes()); diff != "" {
				t.Errorf("processed file data had diffs: %s", diff)
			}
			if diff := cmp.Diff(tt.wantAddCycle, tt.fakeCycles.AddedCycle); diff != "" {
				t.Errorf("added cycle differs: %v", diff)
			}
		})
	}
}
