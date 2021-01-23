// Copyright 2021 Google LLC
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

package smskeys_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/google/exposure-notifications-verification-server/internal/envstest"
	"github.com/google/exposure-notifications-verification-server/internal/project"
	"github.com/google/exposure-notifications-verification-server/pkg/config"
	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/controller/smskeys"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
	"github.com/google/exposure-notifications-verification-server/pkg/keyutils"
	"github.com/google/exposure-notifications-verification-server/pkg/rbac"
	"github.com/google/exposure-notifications-verification-server/pkg/render"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

func TestHandleDestroy(t *testing.T) {
	t.Parallel()

	ctx := project.TestContext(t)
	harness := envstest.NewServer(t, testDatabaseInstance)

	h, err := render.New(ctx, envstest.ServerAssetsPath(), true)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.ServerConfig{}

	t.Run("middleware", func(t *testing.T) {
		t.Parallel()

		publicKeyCache, err := keyutils.NewPublicKeyCache(ctx, harness.Cacher, cfg.CertificateSigning.PublicKeyCacheDuration)
		if err != nil {
			t.Fatal(err)
		}
		c := smskeys.New(ctx, cfg, harness.Database, publicKeyCache, h)
		handler := c.HandleDestroy()

		envstest.ExerciseSessionMissing(t, handler)
		envstest.ExerciseMembershipMissing(t, handler)
		envstest.ExercisePermissionMissing(t, handler)
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()

		publicKeyCache, err := keyutils.NewPublicKeyCache(ctx, harness.Cacher, cfg.CertificateSigning.PublicKeyCacheDuration)
		if err != nil {
			t.Fatal(err)
		}
		c := smskeys.New(ctx, cfg, harness.Database, publicKeyCache, h)
		handler := c.HandleDestroy()

		realm, err := harness.Database.FindRealm(1)
		if err != nil {
			t.Fatal(err)
		}
		// Create 2 signing keys - we need to destroy the non-active one.
		if _, err := realm.CreateSMSSigningKeyVersion(ctx, harness.Database); err != nil {
			t.Fatal(err)
		}
		if _, err := realm.CreateSMSSigningKeyVersion(ctx, harness.Database); err != nil {
			t.Fatal(err)
		}

		keys, err := realm.ListSMSSigningKeys(harness.Database)
		if err != nil {
			t.Fatal(err)
		}
		toDestroy := uint(0)
		for _, k := range keys {
			if !k.Active {
				toDestroy = k.ID
			}
		}

		ctx := ctx
		ctx = controller.WithSession(ctx, &sessions.Session{})
		ctx = controller.WithMembership(ctx, &database.Membership{
			Realm:       realm,
			User:        &database.User{},
			Permissions: rbac.SettingsWrite,
		})

		r := httptest.NewRequest("PUT", "/", nil)
		r = r.Clone(ctx)
		r = mux.SetURLVars(r, map[string]string{"id": fmt.Sprintf("%d", toDestroy)})
		r.Header.Set("Content-Type", "text/html")

		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)
		w.Flush()

		if got, want := w.Code, 303; got != want {
			t.Errorf("expected %d to be %d", got, want)
		}
		if got, want := w.Header().Get("Location"), "/realm/sms-keys"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}

		updatedRealm, err := harness.Database.FindRealm(realm.ID)
		if err != nil {
			t.Fatal(err)
		}

		keys, err = updatedRealm.ListSMSSigningKeys(harness.Database)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 1 {
			t.Fatalf("no SMS key present after create")
		}
		if keys[0].ID == toDestroy {
			t.Fatalf("didn't destroy specified key")
		}
	})
}