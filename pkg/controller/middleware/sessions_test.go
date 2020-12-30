// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/exposure-notifications-verification-server/internal/envstest"
	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/controller/middleware"
	"github.com/google/exposure-notifications-verification-server/pkg/render"
	"github.com/gorilla/sessions"
)

func TestRequireSession(t *testing.T) {
	t.Parallel()

	store := sessions.NewCookieStore()

	h, err := render.New(context.Background(), envstest.ServerAssetsPath(), true)
	if err != nil {
		t.Fatal(err)
	}

	requireSession := middleware.RequireSession(store, h)

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept", "application/json")

	w := httptest.NewRecorder()

	// Verify the proper fields are added to the template map.
	requireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session := controller.SessionFromContext(ctx)
		if session == nil {
			t.Fatalf("expected session")
		}
	})).ServeHTTP(w, r)
}