package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type RoundTripFunc func(*http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONTORemote(t *testing.T) {
	var testTools Tools

	client := NewTestClient(func(*http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"
	_, _, err := testTools.PushJSONTORemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url: ", err.Error())
	}
}

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	if s := testTools.RandomString(10); len(s) != 10 {
		t.Error("Wrong length returned.")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_Uploads(t *testing.T) {
	for _, e := range uploadTests {
		pr, pw := io.Pipe()
		defer pr.Close()
		defer pw.Close()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()
			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools

		testTools.AllowedFileType = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}
		if !e.errorExpected {
			if _, err := os.Stat(filepath.Join("./testdata/uploads/", uploadedFiles[0].NewFileName)); err != nil {
				t.Errorf("%s: expected file to exist:%s", e.name, err.Error())
			}

			//cleanup
			os.Remove(filepath.Join("./testdata/uploads/", uploadedFiles[0].NewFileName))
		}
		if e.errorExpected && err == nil {
			t.Error(err.Error())
		}
		wg.Wait()
	}
}

func TestTools_UploadOne(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads")
	if err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join("./testdata/uploads/", uploadedFiles.NewFileName)); err != nil {
		t.Errorf("%s: expected file to exist", err.Error())
	}

	//cleanup
	os.Remove(filepath.Join("./testdata/uploads/", uploadedFiles.NewFileName))
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools
	err := testTools.CreateDirIfNotExist("./testDir")
	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExist("./testDir")
	if err != nil {
		t.Error(err)
	}

	//cleanup
	os.Remove("./testDir")
}

var slugifyTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "hello there", expected: "hello-there", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "hello there 123 $%^!&&*&!%^!", expected: "hello-there-123", errorExpected: false},
	{name: "non-english string", s: "こんにちは", expected: "", errorExpected: true},
	{name: "roman + non-english string", s: "hello there こんにちは 1234", expected: "hello-there-1234", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var tools Tools
	for _, e := range slugifyTests {
		s, err := tools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s:No error expected but got %s", e.name, err.Error())
		}
		if !e.errorExpected && e.expected != s {
			t.Errorf("%s: Expected %s but got %s", e.name, e.expected, s)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	var tools Tools
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	tools.DownloadStaticFile(rr, req, "./testdata/pic.jpg", "love.jpg")
	res := rr.Result()
	defer res.Body.Close()
	if res.Header["Content-Length"][0] != "61096" {
		t.Error("Expected the soze to be ", res.Header["Content-Length"][0])
	}
	fmt.Println(res.Header["Content-Disposition"][0])
	if res.Header["Content-Disposition"][0] != "attachment; filename\"love.jpg\"" {
		t.Error("wrong content-disposition")
	}
}

var jsonTests = []struct {
	name           string
	json           string
	errorExpected  bool
	maxSize        int
	allowedUnknown bool
}{
	{name: "good json", json: `{"foo":"bar"}`, errorExpected: false, maxSize: 1024, allowedUnknown: false},
	{name: "badly formatted json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowedUnknown: false},
	{name: "incorrect type", json: `{"foo":1}`, errorExpected: true, maxSize: 1024, allowedUnknown: false},
	{name: "two json files", json: `{"foo":"bar"}{"alpha":"beta"}`, errorExpected: true, maxSize: 1024, allowedUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowedUnknown: false},
	{name: "syntax error in json", json: `{"foo":"bar}`, errorExpected: true, maxSize: 1024, allowedUnknown: false},
	{name: "unknown field in json", json: `{"fooo":"bar"}`, errorExpected: true, maxSize: 1024, allowedUnknown: false},
	{name: "allow unknown field in json", json: `{"fooo":"bar"}`, errorExpected: false, maxSize: 1024, allowedUnknown: true},
	{name: "missing field name", json: `{jack:"bar"}`, errorExpected: true, maxSize: 1024, allowedUnknown: true},
	{name: "file too large", json: `{"fooo":"bar"}`, errorExpected: true, maxSize: 5, allowedUnknown: false},
	{name: "not json", json: `hello, world!`, errorExpected: true, maxSize: 1024, allowedUnknown: false},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools
	for _, e := range jsonTests {
		testTools.MaxFileSize = e.maxSize
		testTools.AllowUnknownFields = e.allowedUnknown
		var decodedJSON struct {
			Foo string `json:"foo"`
		}
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log(err)
		}
		rr := httptest.NewRecorder()
		err = testTools.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s : error expected but none received", e.name)
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s : no error expected but got %s", e.name, err.Error())
		}
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}
	headers := make(http.Header)
	headers.Add("foo", "bar")
	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("Failed to write json %v", err)
	}

}

func TestTools_ErrorJson(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	err := testTools.ErrorJson(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}
	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	fmt.Println(payload)
	if err != nil {
		t.Error(err)
	}
	if !payload.Error {
		t.Errorf("error set to false in json; it should be true")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code; expected 503 but got %d", rr.Code)
	}
}
