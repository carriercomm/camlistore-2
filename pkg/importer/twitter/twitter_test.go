/*
Copyright 2014 The Camlistore Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package twitter

import (
	"net/http"
	"path/filepath"
	"testing"

	"camlistore.org/pkg/context"
	"camlistore.org/pkg/test"

	"camlistore.org/third_party/github.com/garyburd/go-oauth/oauth"
)

func TestGetUserID(t *testing.T) {
	ctx := context.New()
	ctx.SetHTTPClient(&http.Client{
		Transport: test.NewFakeTransport(map[string]func() *http.Response{
			apiURL + userInfoAPIPath: test.FileResponder(filepath.FromSlash("testdata/verify_credentials-res.json")),
		}),
	})
	inf, err := getUserInfo(oauthContext{ctx, &oauth.Client{}, &oauth.Credentials{}})
	if err != nil {
		t.Fatal(err)
	}
	want := userInfo{
		ID:         "2325935334",
		ScreenName: "lejatorn",
		Name:       "Mathieu Lonjaret",
	}
	if inf != want {
		t.Errorf("user info = %+v; want %+v", inf, want)
	}
}
