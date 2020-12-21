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

package apikey_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/exposure-notifications-verification-server/internal/browser"
	"github.com/google/exposure-notifications-verification-server/internal/envstest"
	"github.com/google/exposure-notifications-verification-server/pkg/controller/apikey"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
	"github.com/google/exposure-notifications-verification-server/pkg/rbac"
	"github.com/google/exposure-notifications-verification-server/pkg/render"
)

func TestHandleIndex(t *testing.T) {
	t.Parallel()

	harness := envstest.NewServer(t, testDatabaseInstance)

	realm, user, session, err := harness.ProvisionAndLogin()
	if err != nil {
		t.Fatal(err)
	}

	cookie, err := harness.SessionCookie(session)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("middleware", func(t *testing.T) {
		t.Parallel()

		h, err := render.New(context.Background(), envstest.ServerAssetsPath(), true)
		if err != nil {
			t.Fatal(err)
		}
		c := apikey.New(harness.Cacher, harness.Database, h)
		handler := c.HandleIndex()

		envstest.ExerciseMembershipMissing(t, handler)
		envstest.ExercisePermissionMissing(t, handler)
		envstest.ExerciseBadPagination(t, &database.Membership{
			Realm:       realm,
			User:        user,
			Permissions: rbac.APIKeyRead,
		}, handler)
	})

	t.Run("lists", func(t *testing.T) {
		t.Parallel()

		authApp := &database.AuthorizedApp{
			RealmID: realm.ID,
			Name:    "Appy",
		}
		if _, err := realm.CreateAuthorizedApp(harness.Database, authApp, database.SystemTest); err != nil {
			t.Fatal(err)
		}

		browserCtx := browser.New(t)
		taskCtx, done := context.WithTimeout(browserCtx, 10*time.Second)
		defer done()

		if err := chromedp.Run(taskCtx,
			browser.SetCookie(cookie),
			chromedp.Navigate(`http://`+harness.Server.Addr()+`/realm/apikeys`),
			chromedp.WaitVisible(`body#apikeys-index`, chromedp.ByQuery),
			chromedp.WaitVisible(fmt.Sprintf(`tr#apikey-%d`, authApp.ID), chromedp.ByQuery),
		); err != nil {
			t.Fatal(err)
		}
	})
}