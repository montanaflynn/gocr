package gocr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/theckman/yacspin"
)

// BaseURL is the OCR API to use
var BaseURL = "https://api.mathpix.com/v3"

// Client is used to call the Mathpix OCR API
type Client struct {
	http    *http.Client
	spinner *yacspin.Spinner
	appKey  string
}

// pdfResponse gets the pdf_id to query against
type pdfResponse struct {
	ID string `json:"pdf_id"`
}

// Data holds a PDF OCR ID and it's metadata along with methods to
// get the ocr content in a variety of formats
type data struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	NumPages          int     `json:"num_pages"`
	NumPagesCompleted int     `json:"num_pages_completed"`
	PercentDone       float64 `json:"percent_done"`
}

// NewClient creates a ocr client instance
func NewClient(appKey string) *Client {
	return &Client{
		http:   http.DefaultClient,
		appKey: appKey,
	}
}

// SetSpinner sets an optional spinner to show progress
func (c *Client) SetSpinner(spinner *yacspin.Spinner) {
	c.spinner = spinner
}

// Convert PDF to MMD
func (c *Client) Convert(source, destination string) error {
	outputExtension, err := validateExtension(destination)
	if err != nil {
		return err
	}
	id, err := c.uploadFile(source)
	if err != nil {
		return err
	}
	return c.save(id, destination, outputExtension[1:])
}

func validateExtension(path string) (string, error) {
	var validExtensions = []string{".mmd", ".docx", ".tex", ".zip"}
	var pathExtension = filepath.Ext(path)
	for _, validExtension := range validExtensions {
		if pathExtension == validExtension {
			if pathExtension == ".zip" {
				pathExtension = ".tex"
			}
			return pathExtension, nil
		}

	}
	return "", fmt.Errorf("%s does not end with one of: %s", path, validExtensions)
}

func (c *Client) save(ID, path, format string) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/pdf/%s.%s", BaseURL, ID, format), nil)
	if err != nil {
		return err
	}

	req.Header.Set("app_key", c.appKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	if strings.HasSuffix(path, "tex") {
		path = fmt.Sprintf("%s.zip", path)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	io.Copy(f, res.Body)
	return nil
}

func (c *Client) uploadFile(fileName string) (string, error) {
	f, err := os.Open(fileName)
	defer f.Close()

	res, err := c.upload(f.Name(), f)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", errors.New(res.Status)
	}

	decoder := json.NewDecoder(res.Body)
	pdf := &pdfResponse{}
	err = decoder.Decode(pdf)
	if err != nil {
		return "", err
	}

	for {
		time.Sleep(time.Millisecond * 500)
		Data, err := c.getResult(pdf.ID)
		if err != nil {
			return "", err
		}

		if c.spinner != nil {
			c.spinner.Message(fmt.Sprintf(" Processing %.02f%%", Data.PercentDone))
		}

		if Data.PercentDone == 100 {
			return pdf.ID, nil
		}
	}
}

func (c *Client) upload(name string, input io.Reader) (*http.Response, error) {
	body, writer := io.Pipe()
	mwriter := multipart.NewWriter(writer)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", BaseURL, "pdf-file"), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("app_key", c.appKey)
	req.Header.Add("Content-Type", mwriter.FormDataContentType())

	errchan := make(chan error)

	go func() {
		defer close(errchan)
		defer writer.Close()
		defer mwriter.Close()

		w, err := mwriter.CreateFormFile("file", name)
		if err != nil {
			errchan <- err
			return
		}

		if written, err := io.Copy(w, input); err != nil {
			errchan <- fmt.Errorf("error copying %s (%d bytes written): %v", "path", written, err)
			return
		}

		if err := mwriter.Close(); err != nil {
			errchan <- err
			return
		}
	}()

	res, err := c.http.Do(req)
	merr := <-errchan
	if err != nil || merr != nil {
		return res, fmt.Errorf("http error: %v, multipart error: %v", err, merr)
	}

	return res, nil
}

func (c *Client) getResult(id string) (*data, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/pdf/%s", BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("app_key", c.appKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	decoder := json.NewDecoder(res.Body)
	data := &data{}
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
