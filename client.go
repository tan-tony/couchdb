// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package couchdb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kivik/couchdb/v4/chttp"
	kivik "github.com/go-kivik/kivik/v4"
	"github.com/go-kivik/kivik/v4/driver"
)

func (c *client) AllDBs(ctx context.Context, opts map[string]interface{}) ([]string, error) {
	query, err := optionsToParams(opts)
	if err != nil {
		return nil, err
	}
	var allDBs []string
	_, err = c.DoJSON(ctx, http.MethodGet, "/_all_dbs", &chttp.Options{Query: query}, &allDBs)
	return allDBs, err
}

func (c *client) DBExists(ctx context.Context, dbName string, _ map[string]interface{}) (bool, error) {
	if dbName == "" {
		return false, missingArg("dbName")
	}
	_, err := c.DoError(ctx, http.MethodHead, dbName, nil)
	if kivik.StatusCode(err) == http.StatusNotFound {
		return false, nil
	}
	return err == nil, err
}

func (c *client) CreateDB(ctx context.Context, dbName string, opts map[string]interface{}) error {
	if dbName == "" {
		return missingArg("dbName")
	}
	query, err := optionsToParams(opts)
	if err != nil {
		return err
	}
	// change: escape +
	_, err = c.DoError(ctx, http.MethodPut, url.QueryEscape(dbName), &chttp.Options{Query: query})
	return err
}

func (c *client) DestroyDB(ctx context.Context, dbName string, _ map[string]interface{}) error {
	if dbName == "" {
		return missingArg("dbName")
	}
	// change: escape +
	_, err := c.DoError(ctx, http.MethodDelete, url.QueryEscape(dbName), nil)
	return err
}

func (c *client) DBUpdates(ctx context.Context) (updates driver.DBUpdates, err error) {
	resp, err := c.DoReq(ctx, http.MethodGet, "/_db_updates?feed=continuous&since=now", nil)
	if err != nil {
		return nil, err
	}
	if err := chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newUpdates(ctx, resp.Body), nil
}

type couchUpdates struct {
	*iter
}

var _ driver.DBUpdates = &couchUpdates{}

type updatesParser struct{}

var _ parser = &updatesParser{}

func (p *updatesParser) decodeItem(i interface{}, dec *json.Decoder) error {
	return dec.Decode(i)
}

func newUpdates(ctx context.Context, body io.ReadCloser) *couchUpdates {
	return &couchUpdates{
		iter: newIter(ctx, nil, "", body, &updatesParser{}),
	}
}

func (u *couchUpdates) Next(update *driver.DBUpdate) error {
	return u.iter.next(update)
}

// Ping queries the /_up endpoint, and returns true if there are no errors, or
// if a 400 (Bad Request) is returned, and the Server: header indicates a server
// version prior to 2.x.
func (c *client) Ping(ctx context.Context) (bool, error) {
	resp, err := c.DoError(ctx, http.MethodHead, "/_up", nil)
	if kivik.StatusCode(err) == http.StatusBadRequest {
		return strings.HasPrefix(resp.Header.Get("Server"), "CouchDB/1."), nil
	}
	if kivik.StatusCode(err) == http.StatusNotFound {
		return false, nil
	}
	return err == nil, err
}
