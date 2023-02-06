package toolkit

import (
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

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
