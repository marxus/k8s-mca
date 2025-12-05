// Package webhook tests admission webhook mutation and request handling.
package webhook

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewServer(t *testing.T) {
	cert := tls.Certificate{
		Certificate: [][]byte{{1, 2, 3}},
	}

	server := NewServer(cert)

	require.NotNil(t, server)
	assert.Equal(t, cert, server.tlsCert)
}

func TestServer_HandleHealth(t *testing.T) {
	cert := tls.Certificate{}
	server := NewServer(cert)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	server.handleHealth(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "OK", recorder.Body.String())
}

func TestServer_HandleMutate(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    []byte
		wantStatusCode int
		wantAllowed    bool
		wantErrMsg     string
	}{
		{
			name: "valid admission request",
			requestBody: func() []byte {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "nginx",
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "kube-api-access",
										MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
									},
								},
							},
						},
					},
				}
				podJSON, _ := json.Marshal(pod)
				admissionReview := admissionv1.AdmissionReview{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "admission.k8s.io/v1",
						Kind:       "AdmissionReview",
					},
					Request: &admissionv1.AdmissionRequest{
						UID: types.UID("test-uid"),
						Object: runtime.RawExtension{
							Raw: podJSON,
						},
					},
				}
				body, _ := json.Marshal(admissionReview)
				return body
			}(),
			wantStatusCode: http.StatusOK,
			wantAllowed:    true,
		},
		{
			name:           "invalid JSON",
			requestBody:    []byte("invalid json"),
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "Failed to unmarshal admission review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := tls.Certificate{}
			server := NewServer(cert)

			req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(tt.requestBody))
			recorder := httptest.NewRecorder()

			server.handleMutate(recorder, req)

			assert.Equal(t, tt.wantStatusCode, recorder.Code)

			if tt.wantStatusCode == http.StatusOK {
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

				var responseReview admissionv1.AdmissionReview
				err := json.Unmarshal(recorder.Body.Bytes(), &responseReview)
				require.NoError(t, err)

				assert.NotNil(t, responseReview.Response)
				assert.Equal(t, tt.wantAllowed, responseReview.Response.Allowed)
				assert.Equal(t, types.UID("test-uid"), responseReview.Response.UID)
				assert.NotNil(t, responseReview.Response.PatchType)
				assert.NotEmpty(t, responseReview.Response.Patch)
			} else {
				assert.Contains(t, recorder.Body.String(), tt.wantErrMsg)
			}
		})
	}
}

func TestServer_Mutate(t *testing.T) {
	tests := []struct {
		name        string
		podRaw      []byte
		wantAllowed bool
		wantErrMsg  string
	}{
		{
			name: "successfully mutates valid pod",
			podRaw: func() []byte {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "nginx",
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "kube-api-access",
										MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
									},
								},
							},
						},
					},
				}
				podJSON, _ := json.Marshal(pod)
				return podJSON
			}(),
			wantAllowed: true,
		},
		{
			name:        "fails on invalid pod JSON",
			podRaw:      []byte("invalid pod json"),
			wantAllowed: false,
			wantErrMsg:  "Failed to unmarshal pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			admissionReview := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: types.UID("test-uid"),
					Object: runtime.RawExtension{
						Raw: tt.podRaw,
					},
				},
			}

			cert := tls.Certificate{}
			server := NewServer(cert)

			response := server.mutate(admissionReview)

			require.NotNil(t, response)
			assert.Equal(t, "admission.k8s.io/v1", response.APIVersion)
			assert.Equal(t, "AdmissionReview", response.Kind)
			require.NotNil(t, response.Response)
			assert.Equal(t, tt.wantAllowed, response.Response.Allowed)
			assert.Equal(t, types.UID("test-uid"), response.Response.UID)

			if tt.wantAllowed {
				assert.NotNil(t, response.Response.PatchType)
				assert.Equal(t, admissionv1.PatchTypeJSONPatch, *response.Response.PatchType)
				assert.NotEmpty(t, response.Response.Patch)
			} else {
				assert.Contains(t, response.Response.Result.Message, tt.wantErrMsg)
			}
		})
	}
}

func TestServer_GenerateJSONPatch(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
				},
			},
		},
	}

	cert := tls.Certificate{}
	server := NewServer(cert)

	patch, err := server.generateJSONPatch(pod)
	require.NoError(t, err)
	assert.NotEmpty(t, patch)

	var patchOps []map[string]interface{}
	err = json.Unmarshal(patch, &patchOps)
	require.NoError(t, err)

	require.Len(t, patchOps, 1)
	assert.Equal(t, "replace", patchOps[0]["op"])
	assert.Equal(t, "/spec", patchOps[0]["path"])
	assert.NotNil(t, patchOps[0]["value"])
}
