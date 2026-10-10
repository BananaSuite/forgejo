package integration

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	ansible_module "forgejo.org/modules/packages/ansible"
	"forgejo.org/tests"

	gouuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageAnsible(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	token := "Bearer " + getUserToken(t, user.Name, auth_model.AccessTokenScopeWritePackage)

	rootURL := fmt.Sprintf("/api/packages/%s/ansible", user.Name)

	packageNamespace := "forgejo"
	packageName := "test"
	packageVersion := "1.2.3"
	packageAuthors := "Forgejo Authors <authors@forgejo.org>"
	packageDescription := "Test Collection"
	packageRequiredAnsible := ">=2.18.0"
	packageDownloadURL := ""

	// Just some sample collection data
	base64AnsiblePackageContent := `H4sICPmpEGoC/2Zvcmdlam8tdGVzdC0xLjIuMy50YXIA7ZhdT9swFIZz3V8R5YabEWwntlM0EGgr
GtKYpsHdNFWO7dCwfHRxMoEq/vvsfPQD2Hazdmz4vUny2jnnpO15HPfi9MP52eTyyr9RZeFsR0CL
hGF71Hp4hDDADsQIhBCFlEIHQO0BxwXODtSomlW6FOdlajFyPV5mmeR1WhbTtEhK79DVrusVLJdq
zrjUhpeU1bW8Kb1Xw4gxa6nqzvkuK6XvNyb0kR90LmvqWVkp7X7Wl6531gVxTzvffd1POOmj+/pw
7OmpX9rbK8lEl+jT5PTtxcTPRRe3Ztdt0G6akIpX6bzu81/potw3y0fq7shSLgsll5VcnF+t8vSD
0yTNzIyiybI+8FwWQhY8lSbd4r6val6qtC6ru425JW9yWdSsL2M5MCtzOWfXG4FTpZo2pHFGronr
meTTnBVpousfSlmsf9pn5+8nl22fds+U1Hfz7rsxk1uLz76qJp8OA2rGECYbQ72lBymQAgmIEw4p
xGMiY57EMRIxxARIAjEX2iUCc0IFkAkNQSTjAHAWxRTLpC+irHJW63hweI6Vce9YPXetflTby/Eb
/gNCg03+A4oxsvzfFf8NPwZIL1o8Dsjx2yZfQ41Iq97aRM2AtsecWY2so6JlxYNkuazZrvMdVE1R
p7n07/LsYe4lVn/F1SfAihBgfEyCUAAoI6xBCgSinMQMRwkbUxISJiETwTgQOIlCghOYEI5jASIO
Q+A9Vf6oXav+PF39A//g5CO7fafXWlltr/+f6Ptl/4MgXJ0bHwIEdf/f2v7fulDk5qYBjiClUYgQ
AaGPCcYaxSO7Ov7/aiG45RymqfWK/vP+1+dm/de7Pqhx6Zg3AgQdF9v+37qeB/+Dx/yHlv874T9d
8X8cUjImkR+N9VqALf5fDP/XXoL/xv5Pb/xW+z9s/v+DUL8S2v3fDrS/vz+q5LcmraSaskKlcSYP
3b3jI+TDyAd7lgJWVlZWVlZWVlZWVlZWVlZW/7Z+AGRcUvwAKAAA`

	ansiblePackageContent, err := base64.StdEncoding.DecodeString(base64AnsiblePackageContent)
	require.NoError(t, err)

	t.Run("Retrieve API listing", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequest(t, "GET", rootURL)
		resp := MakeRequest(t, req, http.StatusOK)

		var rawJSON any
		DecodeJSON(t, resp, &rawJSON)
		respJSON := rawJSON.(map[string]any)
		rawVersions, ok := respJSON["available_versions"]
		require.True(t, ok)
		versions := rawVersions.(map[string]any)
		v3data, ok := versions["v3"]
		require.True(t, ok)
		assert.Equal(t, "v3/", v3data)
	})

	t.Run("Retrieve v3 root", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequest(t, "GET", rootURL+"/v3/")
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("Retrieve non existing package", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequest(t, "GET", fmt.Sprintf("%s/v3/collections/%s/%s/", rootURL, packageNamespace, packageName))
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("Check import task", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		uuid, _ := gouuid.NewRandom()
		req := NewRequest(t, "GET", fmt.Sprintf("%s/v3/imports/collections/%s", rootURL, uuid.String()))
		resp := MakeRequest(t, req, http.StatusOK)

		var rawJSON any
		DecodeJSON(t, resp, &rawJSON)
		respJSON := rawJSON.(map[string]any)
		responseState, ok := respJSON["state"]
		require.True(t, ok)
		assert.Equal(t, "completed", responseState)
	})

	t.Run("Upload Unauthorized", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequestWithBody(t, "POST", rootURL+"/v3/artifacts/collections", strings.NewReader(""))
		MakeRequest(t, req, http.StatusUnauthorized)
	})

	t.Run("Upload Wrong Method", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequestWithBody(t, "PUT", rootURL+"/v3/artifacts/collections", strings.NewReader(""))
		MakeRequest(t, req, http.StatusMethodNotAllowed)
	})

	// We capture the output of this test, to make sure we only do the subsequent tests if this one is successful
	uploadSuccessful := t.Run("Upload Authorized", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", fmt.Sprintf("%s-%s-%s.tar.gz", packageNamespace, packageName, packageVersion))
		_, _ = io.Copy(part, bytes.NewReader(ansiblePackageContent))
		_ = writer.Close()

		req := NewRequestWithBody(t, "POST", rootURL+"/v3/artifacts/collections", body).
			SetHeader("Content-Type", writer.FormDataContentType()).
			AddTokenAuth(token)

		MakeRequest(t, req, http.StatusCreated)

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeAnsible)
		require.NoError(t, err)
		assert.Len(t, pvs, 1)

		pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
		require.NoError(t, err)
		assert.NotNil(t, pd.SemVer)
		assert.Equal(t, fmt.Sprintf("%s.%s", packageNamespace, packageName), pd.Package.Name)
		assert.Equal(t, packageVersion, pd.Version.Version)
		assert.IsType(t, &ansible_module.CollectionInfo{}, pd.Metadata)
		collection := pd.Metadata.(*ansible_module.CollectionInfo)
		assert.Equal(t, packageNamespace, collection.Namespace)
		assert.Equal(t, packageName, collection.Name)
		assert.Equal(t, packageVersion, collection.Version)
		assert.Len(t, collection.Authors, 1)
		assert.Equal(t, packageAuthors, collection.Authors[0])
		assert.Equal(t, packageDescription, collection.Description)
		assert.Equal(t, packageRequiredAnsible, collection.RequiresAnsible)

		pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
		require.NoError(t, err)
		assert.Len(t, pfs, 1)
		assert.Equal(t, fmt.Sprintf("%s-%s-%s.tar.gz", packageNamespace, packageName, packageVersion), pfs[0].Name)
		assert.True(t, pfs[0].IsLead)
	})

	// Subsequent test assume that the package upload happened, so they are exit early if that didn't happen
	t.Run("Identical upload", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		require.True(t, uploadSuccessful, "Upload from previous test failed.")

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", fmt.Sprintf("%s-%s-%s.tar.gz", packageNamespace, packageName, packageVersion))
		_, _ = io.Copy(part, bytes.NewReader(ansiblePackageContent))
		_ = writer.Close()

		req := NewRequestWithBody(t, "POST", rootURL+"/v3/artifacts/collections", body).
			SetHeader("Content-Type", writer.FormDataContentType()).
			AddTokenAuth(token)

		MakeRequest(t, req, http.StatusConflict)
	})

	t.Run("Retrieve uploaded package metadata", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		require.True(t, uploadSuccessful, "Upload from previous test failed.")

		req := NewRequest(t, "GET", fmt.Sprintf("%s/v3/collections/%s/%s/", rootURL, packageNamespace, packageName))
		resp := MakeRequest(t, req, http.StatusOK)

		var rawJSON any
		DecodeJSON(t, resp, &rawJSON)
		respJSON := rawJSON.(map[string]any)

		assert.Equal(t, packageNamespace, respJSON["namespace"])
		assert.Equal(t, packageName, respJSON["name"])
		assert.Equal(t, fmt.Sprintf("%s/v3/collections/%s/%s/versions/", rootURL, packageNamespace, packageName), respJSON["versions_url"])
		responseHighestVersion := respJSON["highest_version"].(map[string]any)
		assert.Equal(t, packageVersion, responseHighestVersion["version"])
		assert.Equal(t, fmt.Sprintf("%s/v3/collections/%s/%s/versions/%s/", rootURL, packageNamespace, packageName, packageVersion), responseHighestVersion["href"])
	})

	t.Run("Retrieve uploaded package versions", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		require.True(t, uploadSuccessful, "Upload from previous test failed.")

		req := NewRequest(t, "GET", fmt.Sprintf("%s/v3/collections/%s/%s/versions/", rootURL, packageNamespace, packageName))
		resp := MakeRequest(t, req, http.StatusOK)

		var rawJSON any
		DecodeJSON(t, resp, &rawJSON)
		respJSON := rawJSON.(map[string]any)
		assert.Len(t, respJSON["data"], 1)
		version := respJSON["data"].([]any)[0].(map[string]any)
		assert.Equal(t, fmt.Sprintf("%s/v3/collections/%s/%s/versions/%s/", rootURL, packageNamespace, packageName, packageVersion), version["href"])
		assert.Equal(t, packageVersion, version["version"])
		assert.Equal(t, packageRequiredAnsible, version["requires_ansible"])
	})

	t.Run("Retrieve specific version data", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		require.True(t, uploadSuccessful, "Upload from previous test failed.")

		req := NewRequest(t, "GET", fmt.Sprintf("%s/v3/collections/%s/%s/versions/%s/", rootURL, packageNamespace, packageName, packageVersion))
		resp := MakeRequest(t, req, http.StatusOK)

		var rawJSON any
		DecodeJSON(t, resp, &rawJSON)
		respJSON := rawJSON.(map[string]any)
		assert.Equal(t, fmt.Sprintf("%s/v3/collections/%s/%s/versions/%s/", rootURL, packageNamespace, packageName, packageVersion), respJSON["href"])
		assert.Equal(t, packageRequiredAnsible, respJSON["requires_ansible"])
		assert.Contains(t, respJSON["download_url"], fmt.Sprintf("%s/-/packages/ansible/%s.%s/%s/files/", user.Name, packageNamespace, packageName, packageVersion))
		packageDownloadURL = respJSON["download_url"].(string)

		assert.NotNil(t, respJSON["artifact"])
		respArtifact := respJSON["artifact"].(map[string]any)
		assert.Equal(t, fmt.Sprintf("%s-%s-%s.tar.gz", packageNamespace, packageName, packageVersion), respArtifact["filename"])
		assert.InDelta(t, len(ansiblePackageContent), respArtifact["size"], 0.1)

		assert.NotNil(t, respJSON["namespace"])
		respNamespace := respJSON["namespace"].(map[string]any)
		assert.Equal(t, packageNamespace, respNamespace["name"])

		assert.NotNil(t, respJSON["collection"])
		respCollection := respJSON["collection"].(map[string]any)
		assert.Equal(t, packageName, respCollection["name"])
	})

	// This test requires the download URL from the previous test and fails fast if that is not set
	t.Run("Download collection file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		require.True(t, uploadSuccessful, "Upload from previous test failed.")
		require.NotEmpty(t, packageDownloadURL, "Didn't retrieve package file URL during previous tests")

		req := NewRequest(t, "GET", packageDownloadURL)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, ansiblePackageContent, resp.Body.Bytes())
	})
}
