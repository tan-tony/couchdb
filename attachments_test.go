package couchdb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestPutAttachment(t *testing.T) {
	db := &db{}
	_, err := db.PutAttachment(context.Background(), "", "", "", "", nil)
	testy.StatusError(t, "kivik: docID required", kivik.StatusBadRequest, err)
}

func TestPutAttachmentOpts(t *testing.T) {
	tests := []struct {
		name                     string
		db                       *db
		id, rev, filename, ctype string
		body                     io.Reader
		options                  map[string]interface{}

		newRev string
		status int
		err    string
	}{
		{
			name:   "missing docID",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name: "missing filename",
			id:   "foo", rev: "1-xxx",
			status: kivik.StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name: "missing content type",
			id:   "foo", rev: "1-xxx", filename: "x.jpg",
			status: kivik.StatusBadRequest,
			err:    "kivik: contentType required",
		},
		{
			name: "network error",
			id:   "foo", rev: "1-xxx", filename: "x.jpg", ctype: "image/jpeg",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Put http://example.com/testdb/foo/x.jpg?rev=1-xxx: net error",
		},
		{
			name:     "1.6.1",
			id:       "foo",
			rev:      "1-4c6114c65e295552ab1019e2b046b10e",
			filename: "foo.txt",
			ctype:    "text/plain",
			body:     strings.NewReader("Hello, World!"),
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close() // nolint: errcheck
				if ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type")); ct != "text/plain" {
					return nil, fmt.Errorf("Unexpected Content-Type: %s", ct)
				}
				expectedRev := "1-4c6114c65e295552ab1019e2b046b10e"
				if rev := req.URL.Query().Get("rev"); rev != expectedRev {
					return nil, fmt.Errorf("Unexpected rev: %s", rev)
				}
				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				expected := "Hello, World!"
				if string(body) != expected {
					t.Errorf("Unexpected body:\n%s\n", string(body))
				}
				return &http.Response{
					StatusCode: 201,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"Location":       {"http://localhost:5984/foo/foo/foo.txt"},
						"ETag":           {`"2-8ee3381d24ee4ac3e9f8c1f6c7395641"`},
						"Date":           {"Thu, 26 Oct 2017 20:51:35 GMT"},
						"Content-Type":   {"text/plain; charset=utf-8"},
						"Content-Length": {"66"},
						"Cache-Control":  {"must-revalidate"},
					},
					Body: Body(`{"ok":true,"id":"foo","rev":"2-8ee3381d24ee4ac3e9f8c1f6c7395641"}`),
				}, nil
			}),
			newRev: "2-8ee3381d24ee4ac3e9f8c1f6c7395641",
		},
		{
			name:     "no rev",
			id:       "foo",
			filename: "foo.txt",
			ctype:    "text/plain",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				if _, ok := req.URL.Query()["rev"]; ok {
					t.Errorf("'rev' should not be present in the query")
				}
				return nil, errors.New("ignore this error")
			}),
			status: 601,
			err:    "Put http://example.com/testdb/foo/foo.txt: ignore this error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.PutAttachmentOpts(context.Background(), test.id, test.rev, test.filename, test.ctype, test.body, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Expected %s, got %s\n", test.newRev, newRev)
			}
		})
	}
}

func TestGetAttachmentMeta(t *testing.T) {
	tests := []struct {
		name              string
		id, rev, filename string
		db                *db
		ctype             string
		md5               driver.MD5sum
		status            int
		err               string
	}{
		{
			name:     "network error",
			id:       "foo",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("net error")),
			status:   kivik.StatusNetworkError,
			err:      "Head http://example.com/testdb/foo/foo.txt: net error",
		},
		{
			name:     "1.6.1",
			id:       "foo",
			filename: "foo.txt",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":           {`"gSr8dSmynwAoomH7V6RVYw=="`},
					"Date":           {"Thu, 26 Oct 2017 21:15:13 GMT"},
					"Content-Type":   {"text/plain"},
					"Content-Length": {"13"},
					"Cache-Control":  {"must-revalidate"},
					"Accept-Ranges":  {"none"},
				},
				Body: Body(""),
			}, nil),
			md5:   driver.MD5sum{0x81, 0x2a, 0xfc, 0x75, 0x29, 0xb2, 0x9f, 0x00, 0x28, 0xa2, 0x61, 0xfb, 0x57, 0xa4, 0x55, 0x63},
			ctype: "text/plain",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctype, md5sum, err := test.db.GetAttachmentMeta(context.Background(), test.id, test.rev, test.filename)
			testy.StatusError(t, test.err, test.status, err)
			if ctype != test.ctype {
				t.Errorf("Unexpected Content-Type: %s", ctype)
			}
			if md5sum != test.md5 {
				t.Errorf("Unexpected MD5 Sum: %0x", md5sum)
			}
		})
	}
}

func TestGetMD5Checksum(t *testing.T) {
	tests := []struct {
		name   string
		resp   *http.Response
		sum    driver.MD5sum
		status int
		err    string
	}{
		{
			name:   "no etag header",
			resp:   &http.Response{},
			status: kivik.StatusBadResponse,
			err:    "ETag header not found",
		},
		{
			name: "invalid ETag header",
			resp: &http.Response{
				Header: http.Header{"ETag": []string{`invalid base64`}},
			},
			status: kivik.StatusBadResponse,
			err:    "failed to decode MD5 checksum: illegal base64 data at input byte 7",
		},
		{
			name: "Standard ETag header",
			resp: &http.Response{
				Header: http.Header{"ETag": []string{`"ENGoH7oK8V9R3BMnfDHZmw=="`}},
			},
			sum: driver.MD5sum{0x10, 0xd1, 0xa8, 0x1f, 0xba, 0x0a, 0xf1, 0x5f, 0x51, 0xdc, 0x13, 0x27, 0x7c, 0x31, 0xd9, 0x9b},
		},
		{
			name: "normalized Etag header",
			resp: &http.Response{
				Header: http.Header{"Etag": []string{`"ENGoH7oK8V9R3BMnfDHZmw=="`}},
			},
			sum: driver.MD5sum{0x10, 0xd1, 0xa8, 0x1f, 0xba, 0x0a, 0xf1, 0x5f, 0x51, 0xdc, 0x13, 0x27, 0x7c, 0x31, 0xd9, 0x9b},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sum, err := getMD5Checksum(test.resp)
			testy.Error(t, test.err, err)
			if sum != test.sum {
				t.Errorf("Unexpected result: %0x", sum)
			}
		})
	}
}

func TestGetAttachment(t *testing.T) {
	tests := []struct {
		name              string
		id, rev, filename string
		db                *db
		ctype             string
		md5               driver.MD5sum
		content           string
		status            int
		err               string
	}{
		{
			name:     "network error",
			id:       "foo",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("net error")),
			status:   kivik.StatusNetworkError,
			err:      "Get http://example.com/testdb/foo/foo.txt: net error",
		},
		{
			name:     "1.6.1",
			id:       "foo",
			filename: "foo.txt",
			db: newCustomDB(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
						"ETag":           {`"gSr8dSmynwAoomH7V6RVYw=="`},
						"Date":           {"Fri, 27 Oct 2017 11:24:50 GMT"},
						"Content-Type":   {"text/plain"},
						"Content-Length": {"13"},
						"Cache-Control":  {"must-revalidate"},
						"Accept-Ranges":  {"none"},
					},
					Body: Body(`Hello, world!`),
				}, nil
			}),
			ctype:   "text/plain",
			md5:     driver.MD5sum{0x81, 0x2a, 0xfc, 0x75, 0x29, 0xb2, 0x9f, 0x00, 0x28, 0xa2, 0x61, 0xfb, 0x57, 0xa4, 0x55, 0x63},
			content: "Hello, world!",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctype, md5, content, err := test.db.GetAttachment(context.Background(), test.id, test.rev, test.filename)
			testy.StatusError(t, test.err, test.status, err)
			defer content.Close() // nolint: errcheck
			if ctype != test.ctype {
				t.Errorf("Unexpected content type: %s", ctype)
			}
			if md5 != test.md5 {
				t.Errorf("Unexpected MD5 sum: %0x", md5)
			}
			fileContent, err := ioutil.ReadAll(content)
			if err != nil {
				t.Fatal(err)
			}
			if d := diff.Text(test.content, string(fileContent)); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestFetchAttachment(t *testing.T) {
	tests := []struct {
		name                      string
		method, id, rev, filename string
		db                        *db
		resp                      *http.Response
		status                    int
		err                       string
	}{
		{
			name:   "no method",
			status: kivik.StatusInternalServerError,
			err:    "method required",
		},
		{
			name:   "no docID",
			method: "GET",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no filename",
			method: "GET",
			id:     "foo",
			status: kivik.StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name:     "no rev",
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("ignore this error")),
			status:   601,
			err:      "http://example.com/testdb/foo/foo.txt:",
		},
		{
			name:     "with rev",
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			rev:      "1-xxx",
			db:       newTestDB(nil, errors.New("ignore this error")),
			status:   601,
			err:      "http://example.com/testdb/foo/foo.txt\\?rev=1-xxx:",
		},
		{
			name:     "success",
			method:   "GET",
			id:       "foo",
			filename: "foo.txt",
			rev:      "1-xxx",
			db: newTestDB(&http.Response{
				StatusCode: 200,
			}, nil),
			resp: &http.Response{
				StatusCode: 200,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, err := test.db.fetchAttachment(context.Background(), test.method, test.id, test.rev, test.filename)
			testy.StatusErrorRE(t, test.err, test.status, err)
			resp.Request = nil
			if d := diff.Interface(test.resp, resp); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDecodeAttachment(t *testing.T) {
	tests := []struct {
		name    string
		resp    *http.Response
		ctype   string
		md5     driver.MD5sum
		content string
		status  int
		err     string
	}{
		{
			name:   "no content type",
			resp:   &http.Response{},
			status: kivik.StatusBadResponse,
			err:    "no Content-Type in response",
		},
		{
			name: "no etag header",
			resp: &http.Response{
				Header: http.Header{"Content-Type": {"text/plain"}},
			},
			status: kivik.StatusBadResponse,
			err:    "ETag header not found",
		},
		{
			name: "success",
			resp: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/plain"},
					"ETag":         {`"gSr8dSmynwAoomH7V6RVYw=="`},
				},
				Body: Body("Hello, World!"),
			},
			ctype:   "text/plain",
			md5:     driver.MD5sum{0x81, 0x2a, 0xfc, 0x75, 0x29, 0xb2, 0x9f, 0x00, 0x28, 0xa2, 0x61, 0xfb, 0x57, 0xa4, 0x55, 0x63},
			content: "Hello, World!",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctype, md5, content, err := decodeAttachment(test.resp)
			testy.StatusError(t, test.err, test.status, err)
			if ctype != test.ctype {
				t.Errorf("Unexpected content type: %s", ctype)
			}
			if md5 != test.md5 {
				t.Errorf("Unexpected MD5 sum: %0x", md5)
			}
			fileContent, err := ioutil.ReadAll(content)
			if err != nil {
				t.Fatal(err)
			}
			if d := diff.Text(test.content, string(fileContent)); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDeleteAttachment(t *testing.T) {
	tests := []struct {
		name              string
		id, rev, filename string
		db                *db
		newRev            string
		status            int
		err               string
	}{
		{
			name:   "no doc id",
			status: kivik.StatusBadRequest,
			err:    "kivik: docID required",
		},
		{
			name:   "no rev",
			id:     "foo",
			status: kivik.StatusBadRequest,
			err:    "kivik: rev required",
		},
		{
			name:   "no filename",
			id:     "foo",
			rev:    "1-xxx",
			status: kivik.StatusBadRequest,
			err:    "kivik: filename required",
		},
		{
			name:     "network error",
			id:       "foo",
			rev:      "1-xxx",
			filename: "foo.txt",
			db:       newTestDB(nil, errors.New("net error")),
			status:   kivik.StatusNetworkError,
			err:      "(Delete http://example.com/testdb/foo/foo.txt?rev=1-xxx: )?net error",
		},
		{
			name:     "success 1.6.1",
			id:       "foo",
			rev:      "2-8ee3381d24ee4ac3e9f8c1f6c7395641",
			filename: "foo.txt",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Server":         {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"ETag":           {`"3-231a932924f61816915289fecd35b14a"`},
					"Date":           {"Fri, 27 Oct 2017 13:30:40 GMT"},
					"Content-Type":   {"text/plain; charset=utf-8"},
					"Content-Length": {"66"},
					"Cache-Control":  {"must-revalidate"},
				},
				Body: Body(`{"ok":true,"id":"foo","rev":"3-231a932924f61816915289fecd35b14a"}`),
			}, nil),
			newRev: "3-231a932924f61816915289fecd35b14a",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRev, err := test.db.DeleteAttachment(context.Background(), test.id, test.rev, test.filename)
			testy.StatusErrorRE(t, test.err, test.status, err)
			if newRev != test.newRev {
				t.Errorf("Unexpected new rev: %s", newRev)
			}
		})
	}
}
