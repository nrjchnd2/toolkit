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
