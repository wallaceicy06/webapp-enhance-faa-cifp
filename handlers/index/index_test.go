package index

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/templates"
)

type fakeCyclesLister struct {
	Cycles []*db.Cycle
	Err    error
}

func (fl *fakeCyclesLister) List(context.Context) ([]*db.Cycle, error) {
	return fl.Cycles, fl.Err
}

func TestIndexHandler(t *testing.T) {
	for _, tt := range []struct {
		name           string
		cyclesLister   *fakeCyclesLister
		wantBaseValues *baseValues
	}{
		{
			name: "Good",
			cyclesLister: &fakeCyclesLister{
				Cycles: []*db.Cycle{
					{Name: "first-cycle", ProcessedURL: "http://some-url-1"},
					{Name: "second-cycle", ProcessedURL: "http://some-url-2"},
				},
			},
			wantBaseValues: &baseValues{
				Cycles: []*db.Cycle{
					{Name: "first-cycle", ProcessedURL: "http://some-url-1"},
					{Name: "second-cycle", ProcessedURL: "http://some-url-2"},
				},
			},
		},
		{
			name: "ListError",
			cyclesLister: &fakeCyclesLister{
				Err: errors.New("list error"),
			},
			wantBaseValues: &baseValues{
				DisplayError: "The enhanced FAA CIFP data U/S. We apologize for the inconvenience.",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := &Handler{
				Cycles: tt.cyclesLister,
			}
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf(
					"unexpected status: got (%v) want (%v)",
					status,
					http.StatusOK,
				)
			}

			var expected bytes.Buffer
			if err := templates.Base.Execute(&expected, tt.wantBaseValues); err != nil {
				t.Fatalf("could not execute expected template: %v", err)
			}
			if diff := cmp.Diff(expected.String(), rr.Body.String()); diff != "" {
				t.Errorf("unexpected body diff: %s", diff)
			}
		})
	}
}

func TestIndexHandlerNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/404", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := &Handler{}
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusNotFound,
		)
	}
}
