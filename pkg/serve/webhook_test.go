package serve

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestBuildWebhookPatch(t *testing.T) {
	caCertPEM := []byte("test-certificate-data")

	patch := buildWebhookPatch(caCertPEM)

	// Verify patch is valid JSON
	var patchOps []map[string]interface{}
	err := json.Unmarshal(patch, &patchOps)
	require.NoError(t, err)

	// Verify patch structure
	require.Len(t, patchOps, 1)
	assert.Equal(t, "replace", patchOps[0]["op"])
	assert.Equal(t, "/webhooks/0/clientConfig/caBundle", patchOps[0]["path"])

	// Verify base64 encoded value
	expectedEncoded := base64.StdEncoding.EncodeToString(caCertPEM)
	assert.Equal(t, expectedEncoded, patchOps[0]["value"])

	// Verify we can decode it back
	decoded, err := base64.StdEncoding.DecodeString(patchOps[0]["value"].(string))
	require.NoError(t, err)
	assert.Equal(t, caCertPEM, decoded)
}

func TestBuildWebhookPatch_EmptyCert(t *testing.T) {
	caCertPEM := []byte("")

	patch := buildWebhookPatch(caCertPEM)

	var patchOps []map[string]interface{}
	err := json.Unmarshal(patch, &patchOps)
	require.NoError(t, err)

	assert.Equal(t, "", patchOps[0]["value"])
}

func TestPatchMutatingConfig(t *testing.T) {
	caCertPEM := []byte("test-certificate-data")

	// Create fake clientset
	fakeClient := fake.NewSimpleClientset()

	// Track if patch was called
	patchCalled := false
	var patchedName string
	var patchType types.PatchType
	var patchData []byte

	fakeClient.PrependReactor("patch", "mutatingwebhookconfigurations", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		patchCalled = true
		patchedName = patchAction.GetName()
		patchType = patchAction.GetPatchType()
		patchData = patchAction.GetPatch()
		return true, nil, nil
	})

	err := patchMutatingConfig(caCertPEM, fakeClient)
	require.NoError(t, err)

	// Verify patch was called
	assert.True(t, patchCalled)
	assert.Equal(t, conf.WebhookName, patchedName)
	assert.Equal(t, types.JSONPatchType, patchType)

	// Verify patch data
	var patchOps []map[string]interface{}
	err = json.Unmarshal(patchData, &patchOps)
	require.NoError(t, err)
	require.Len(t, patchOps, 1)
	assert.Equal(t, "replace", patchOps[0]["op"])
	assert.Equal(t, "/webhooks/0/clientConfig/caBundle", patchOps[0]["path"])
}

func TestPatchMutatingConfig_PatchError(t *testing.T) {
	caCertPEM := []byte("test-certificate-data")

	// Create fake clientset that returns error
	fakeClient := fake.NewSimpleClientset()
	fakeClient.PrependReactor("patch", "mutatingwebhookconfigurations", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, assert.AnError
	})

	err := patchMutatingConfig(caCertPEM, fakeClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to patch mutating webhook")
}

func TestStartWebhook_NamespaceFileNotFound(t *testing.T) {
	// Setup empty filesystem
	fs := afero.NewMemMapFs()
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	err := StartWebhook()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read namespace file")
}
