package process

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/wallaceicy06/enhance-faa-cifp/enhance"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
)

type googleVerifier interface {
	VerifyGoogle(context.Context, string) (string, error)
}

type cyclesAdderGetter interface {
	Add(context.Context, *db.Cycle) error
	Get(context.Context, string) (*db.Cycle, error)
}

type Handler struct {
	ServiceAccountEmail string
	Verifier            googleVerifier
	Cycles              cyclesAdderGetter
	DisableAuth         bool
	CifpURL             string
	GetStorageWriter    func(_ context.Context, bucket, fileName string) io.WriteCloser
}

type faaCIFPInfoResponse struct {
	Edition []struct {
		Name    string `json:"editionName"`
		Format  string `json:"format"`
		Date    string `json:"editionDate"`
		Number  int    `json:"editionNumber"`
		Product struct {
			Name string `json:"productName"`
			URL  string `json:"url"`
		} `json:"product"`
	} `json:"edition"`
}

// Handle processes the latest CIFP data and saves it to Google Cloud Storage.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.DisableAuth {
		a := r.Header.Get("Authorization")
		if a == "" {
			http.Error(w, "Must provide credentials.", http.StatusUnauthorized)
			return
		}
		email, err := h.Verifier.VerifyGoogle(r.Context(), auth.ParseAuthHeader(a))
		if err != nil {
			http.Error(w, "Invalid credentials.", http.StatusForbidden)
			return
		}
		log.Printf("got user with email: %q", email)
		if email != h.ServiceAccountEmail {
			http.Error(w, "Invalid credentials.", http.StatusForbidden)
			return
		}
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, h.CifpURL, nil)
	if err != nil {
		log.Printf("Could not create request: %v", err)
		http.Error(w, "Problem making request to FAA.", http.StatusInternalServerError)
		return
	}
	req.Header.Set("accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Could not fetch FAA data: %v", err)
		http.Error(w, "Could not fetch FAA data.", http.StatusInternalServerError)
		return
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("Could not fetch FAA data, got status %s", res.Status)
		http.Error(w, "Could not fetch FAA data.", http.StatusInternalServerError)
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Could not read FAA data body: %v", err)
		http.Error(w, "Could not read FAA data.", http.StatusInternalServerError)
		return
	}
	var resCIFPInfo faaCIFPInfoResponse
	log.Printf("raw: %s", b)
	if err := json.Unmarshal(b, &resCIFPInfo); err != nil {
		log.Printf("Could not unmarshal data: %v", err)
		http.Error(w, "Could not read FAA data.", http.StatusInternalServerError)
		return
	}

	if len(resCIFPInfo.Edition) == 0 {
		log.Printf("Received no editions from FAA, want at least 1.")
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}
	edition := resCIFPInfo.Edition[0]
	c, err := h.Cycles.Get(r.Context(), edition.Date)
	if err != nil {
		log.Printf("Problem getting cycles: %v", err)
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}
	if c != nil {
		log.Printf("Data already processed for %q, skipping.", edition.Date)
		return
	}

	fileReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, edition.Product.URL, nil)
	if err != nil {
		log.Printf("Could not create request: %v", err)
		http.Error(w, "Problem making request to FAA.", http.StatusInternalServerError)
		return
	}
	fileRes, err := http.DefaultClient.Do(fileReq)
	if err != nil {
		log.Printf("Could not fetch CIFP file: %v", err)
		http.Error(w, "Could not fetch FAA data.", http.StatusInternalServerError)
		return
	}
	defer fileRes.Body.Close()
	tmpData, err := ioutil.TempFile("", "tempfaadata.zip")
	if err != nil {
		log.Printf("Could not create temp file: %v", err)
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpData.Name())
	originalName := "original/FAACIFP18_original_" + convertDateToFilename(edition.Date) + ".zip"
	originalWriter := h.GetStorageWriter(r.Context(), "faa-cifp-data", originalName)
	defer func() {
		if err := originalWriter.Close(); err != nil {
			log.Printf("Could not close orignal writer: %v", err)
		}
		log.Printf("Closed original writer.")
	}()
	mw := io.MultiWriter(originalWriter, tmpData)
	bufSize, err := io.Copy(mw, fileRes.Body)
	if err != nil {
		log.Printf("Could not copy data: %v", err)
		http.Error(w, "Could not fetch FAA data.", http.StatusInternalServerError)
		return
	}
	log.Print("Copied original data.")

	zipReader, err := zip.NewReader(tmpData, bufSize)
	if err != nil {
		log.Printf("Could not unzip data: %v", err)
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}
	var cifpZipFileReader io.ReadCloser
	for _, zipFile := range zipReader.File {
		log.Printf("file: %q", zipFile.Name)
		if strings.HasSuffix(zipFile.Name, "FAACIFP18") {
			if cifpZipFileReader, err = zipFile.Open(); err != nil {
				log.Printf("Could not open file from zip archive: %v", err)
				http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
				return
			}
			break
		}
	}
	if cifpZipFileReader == nil {
		log.Print("Could not find FAACIFP18 file in zip archive.")
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}
	defer cifpZipFileReader.Close()
	tmpCifpData, err := ioutil.TempFile("", "tempcifpdata")
	if err != nil {
		log.Printf("Could not create temp file: %v", err)
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpCifpData.Name())
	if _, err := io.Copy(tmpCifpData, cifpZipFileReader); err != nil {
		log.Printf("Could not copy data: %v", err)
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}

	processedName := "processed/FAACIFP18_processed_" + convertDateToFilename(edition.Date)
	processedWriter := h.GetStorageWriter(r.Context(), "faa-cifp-data", processedName)
	defer processedWriter.Close()
	if err := enhance.Process(tmpCifpData, processedWriter, enhance.RemoveDuplicateLocalizers(true)); err != nil {
		log.Printf("Could not process data: %v", err)
		http.Error(w, "Could not process FAA data.", http.StatusInternalServerError)
		return
	}

	if err := h.Cycles.Add(r.Context(), &db.Cycle{
		Name:         resCIFPInfo.Edition[0].Date,
		OriginalURL:  "https://storage.googleapis.com/faa-cifp-data/" + originalName,
		ProcessedURL: "https://storage.googleapis.google.com/faa-cifp-data/" + processedName,
	}); err != nil {
		log.Printf("Could not add cycle: %v", err)
		http.Error(w, "Could not add cycle.", http.StatusInternalServerError)
		return
	}
}

func convertDateToFilename(date string) string {
	return strings.Replace(date, "/", "-", -1)
}
