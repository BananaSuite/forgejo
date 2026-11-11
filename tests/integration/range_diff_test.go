// Copyright 2026 The Forgejo Authors
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"
)

func TestRangeDiff(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")

	t.Run("Success", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		urls := []string{
			"/user2/commitsonpr/range-diff/cbff181af4c9c7fee3cf6c106699e07d9a3f54e6..1978192d98bb1b65e11c2cf37da854fbf94bffd6/cbff181af4c9c7fee3cf6c106699e07d9a3f54e6..9b93963cf6de4dc33f915bb67f192d099c301f43",
			"/user2/commitsonpr/range-diff/cbff181..1978192/cbff181..9b93963", // short refs should still be resolved and render full-length links
		}
		for _, u := range urls {
			req := NewRequest(t, "GET", u)
			resp := session.MakeRequest(t, req, http.StatusOK)
			doc := NewHTMLParser(t, resp.Body)

			// "Comparing commit set from a..b against commit set from c..d"; check that links exist:
			doc.AssertElement(t, ".range-diff-description a:nth-child(1)[href='/user2/commitsonpr/commit/cbff181af4c9c7fee3cf6c106699e07d9a3f54e6']", true)
			doc.AssertElement(t, ".range-diff-description a:nth-child(2)[href='/user2/commitsonpr/commit/1978192d98bb1b65e11c2cf37da854fbf94bffd6']", true)
			doc.AssertElement(t, ".range-diff-description a:nth-child(3)[href='/user2/commitsonpr/commit/cbff181af4c9c7fee3cf6c106699e07d9a3f54e6']", true)
			doc.AssertElement(t, ".range-diff-description a:nth-child(4)[href='/user2/commitsonpr/commit/9b93963cf6de4dc33f915bb67f192d099c301f43']", true)
		}
	})

	t.Run("Malformed URL", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		// "a..b" missing two components in either commit set:
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/commitsonpr/range-diff/cbff181/cbff181..9b93963"), http.StatusBadRequest)
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/commitsonpr/range-diff/cbff181..1978192/cbff181"), http.StatusBadRequest)
	})

	t.Run("Missing Ref", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/commitsonpr/range-diff/bad..1978192/cbff181..9b93963"), http.StatusNotFound)
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/commitsonpr/range-diff/cbff181..bad/cbff181..9b93963"), http.StatusNotFound)
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/commitsonpr/range-diff/cbff181..1978192/bad..9b93963"), http.StatusNotFound)
		session.MakeRequest(t, NewRequest(t, "GET", "/user2/commitsonpr/range-diff/cbff181..1978192/cbff181..bad"), http.StatusNotFound)
	})
}
