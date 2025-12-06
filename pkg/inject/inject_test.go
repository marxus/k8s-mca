// Package inject tests pod mutation for MCA proxy sidecar injection.
package inject

import (
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestViaCLI(t *testing.T) {
	tests := []struct {
		name    string
		podYAML string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pod YAML",
			podYAML: `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: kube-api-access
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
  volumes:
  - name: kube-api-access
    projected:
      sources:
      - serviceAccountToken:
          path: token
`,
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			podYAML: `invalid yaml: {{{`,
			wantErr: true,
			errMsg:  "failed to unmarshal pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ViaCLI([]byte(tt.podYAML))

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)

				var resultPod corev1.Pod
				err = yaml.Unmarshal(result, &resultPod)
				require.NoError(t, err)

				assert.Len(t, resultPod.Spec.InitContainers, 1)
				assert.Equal(t, "mca-proxy", resultPod.Spec.InitContainers[0].Name)
			}
		})
	}
}

func TestViaWebhook_BasicPod(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
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

	result, err := ViaWebhook(pod)
	require.NoError(t, err)

	assert.Len(t, result.Spec.InitContainers, 1)
	assert.Equal(t, "mca-proxy", result.Spec.InitContainers[0].Name)
	assert.Equal(t, conf.ProxyImage, result.Spec.InitContainers[0].Image)
}

func TestInjectProxy_AddsProxyInitContainer(t *testing.T) {
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

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.InitContainers, 1)
	proxyContainer := result.Spec.InitContainers[0]
	assert.Equal(t, "mca-proxy", proxyContainer.Name)
	assert.Equal(t, conf.ProxyImage, proxyContainer.Image)
	assert.Equal(t, []string{"--proxy"}, proxyContainer.Args)
	assert.NotNil(t, proxyContainer.SecurityContext)
	assert.Equal(t, int64(999), *proxyContainer.SecurityContext.RunAsUser)
	assert.True(t, *proxyContainer.SecurityContext.RunAsNonRoot)
}

func TestInjectProxy_PreservesExistingProxyContainer(t *testing.T) {
	existingProxy := corev1.Container{
		Name:  "mca-proxy",
		Image: "custom-proxy:v2",
		Args:  []string{"--custom-arg"},
	}

	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{existingProxy},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
				},
			},
		},
	}

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.InitContainers, 1)
	assert.Equal(t, "mca-proxy", result.Spec.InitContainers[0].Name)
	assert.Equal(t, "custom-proxy:v2", result.Spec.InitContainers[0].Image)
	assert.Equal(t, []string{"--custom-arg"}, result.Spec.InitContainers[0].Args)
}

func TestInjectProxy_PreservesOtherInitContainers(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "init-db",
					Image: "postgres:init",
				},
				{
					Name:  "init-cache",
					Image: "redis:init",
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
				},
			},
		},
	}

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.InitContainers, 3)
	assert.Equal(t, "mca-proxy", result.Spec.InitContainers[0].Name)
	assert.Equal(t, "init-db", result.Spec.InitContainers[1].Name)
	assert.Equal(t, "init-cache", result.Spec.InitContainers[2].Name)
}

func TestInjectProxy_UpdatesVolumeMountAndAddsEnvVars(t *testing.T) {
	pod := corev1.Pod{
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

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.Containers, 1)
	container := result.Spec.Containers[0]

	assert.Equal(t, "kube-api-access-mca-sa", container.VolumeMounts[0].Name)

	require.Len(t, container.Env, 2)
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}
	assert.Equal(t, "127.0.0.1", envMap["KUBERNETES_SERVICE_HOST"])
	assert.Equal(t, "6443", envMap["KUBERNETES_SERVICE_PORT"])
}

func TestInjectProxy_AddsServiceAccountMountToAllContainers(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/data",
						},
					},
				},
			},
		},
	}

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.Containers, 1)
	container := result.Spec.Containers[0]

	// Should have both the original mount and the new MCA serviceaccount mount
	require.Len(t, container.VolumeMounts, 2)
	assert.Equal(t, "data", container.VolumeMounts[0].Name)
	assert.Equal(t, "kube-api-access-mca-sa", container.VolumeMounts[1].Name)
	assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount", container.VolumeMounts[1].MountPath)
	assert.True(t, container.VolumeMounts[1].ReadOnly)

	// Should have env vars added
	require.Len(t, container.Env, 2)
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}
	assert.Equal(t, "127.0.0.1", envMap["KUBERNETES_SERVICE_HOST"])
	assert.Equal(t, "6443", envMap["KUBERNETES_SERVICE_PORT"])
}

func TestInjectProxy_AddsRequiredVolume(t *testing.T) {
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

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.Volumes, 1)
	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Volumes[0].Name)
	assert.NotNil(t, result.Spec.Volumes[0].EmptyDir)
}

func TestInjectProxy_DoesNotDuplicateVolume(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "kube-api-access-mca-sa",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	result, err := injectProxy(pod)
	require.NoError(t, err)

	assert.Len(t, result.Spec.Volumes, 1)
	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Volumes[0].Name)
}

func TestAddVolumeMount(t *testing.T) {
	tests := []struct {
		name               string
		volumeMounts       []corev1.VolumeMount
		wantVolumeMounts   int  // expected number of volume mounts after modification
		wantFirstMountName string // expected first volume mount name
	}{
		{
			name: "updates existing serviceaccount mount",
			volumeMounts: []corev1.VolumeMount{
				{
					Name:      "original-name",
					MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
				},
			},
			wantVolumeMounts:   1,
			wantFirstMountName: "kube-api-access-mca-sa",
		},
		{
			name: "adds mount when no matching mount path exists",
			volumeMounts: []corev1.VolumeMount{
				{
					Name:      "data",
					MountPath: "/data",
				},
			},
			wantVolumeMounts:   2,
			wantFirstMountName: "data",
		},
		{
			name:               "adds mount when volume mounts are empty",
			volumeMounts:       []corev1.VolumeMount{},
			wantVolumeMounts:   1,
			wantFirstMountName: "kube-api-access-mca-sa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &corev1.Container{
				Name:         "app",
				VolumeMounts: tt.volumeMounts,
			}

			addVolumeMount(container)

			assert.Len(t, container.VolumeMounts, tt.wantVolumeMounts)
			if tt.wantVolumeMounts > 0 {
				assert.Equal(t, tt.wantFirstMountName, container.VolumeMounts[0].Name)
				// Verify the MCA mount exists
				found := false
				for _, mount := range container.VolumeMounts {
					if mount.Name == "kube-api-access-mca-sa" &&
						mount.MountPath == "/var/run/secrets/kubernetes.io/serviceaccount" &&
						mount.ReadOnly {
						found = true
						break
					}
				}
				assert.True(t, found, "MCA serviceaccount mount should exist")
			}
		})
	}
}

func TestAddEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		initialEnv  []corev1.EnvVar
		wantEnvLen  int
		wantEnvVars map[string]string // expected final env vars
	}{
		{
			name:       "adds new env vars to empty container",
			initialEnv: []corev1.EnvVar{},
			wantEnvLen: 2,
			wantEnvVars: map[string]string{
				"KUBERNETES_SERVICE_HOST": "127.0.0.1",
				"KUBERNETES_SERVICE_PORT": "6443",
			},
		},
		{
			name: "updates existing env vars",
			initialEnv: []corev1.EnvVar{
				{Name: "KUBERNETES_SERVICE_HOST", Value: "old-value"},
				{Name: "OTHER_VAR", Value: "keep-me"},
			},
			wantEnvLen: 3,
			wantEnvVars: map[string]string{
				"KUBERNETES_SERVICE_HOST": "127.0.0.1",
				"KUBERNETES_SERVICE_PORT": "6443",
				"OTHER_VAR":               "keep-me",
			},
		},
		{
			name: "preserves other env vars",
			initialEnv: []corev1.EnvVar{
				{Name: "APP_ENV", Value: "production"},
				{Name: "DEBUG", Value: "false"},
			},
			wantEnvLen: 4,
			wantEnvVars: map[string]string{
				"APP_ENV":                 "production",
				"DEBUG":                   "false",
				"KUBERNETES_SERVICE_HOST": "127.0.0.1",
				"KUBERNETES_SERVICE_PORT": "6443",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &corev1.Container{
				Name: "app",
				Env:  tt.initialEnv,
			}

			addEnvVars(container)

			require.Len(t, container.Env, tt.wantEnvLen)

			envMap := make(map[string]string)
			for _, env := range container.Env {
				envMap[env.Name] = env.Value
			}

			for key, value := range tt.wantEnvVars {
				assert.Equal(t, value, envMap[key], "env var %s", key)
			}
		})
	}
}

func TestAddRequiredVolume(t *testing.T) {
	tests := []struct {
		name           string
		initialVolumes []corev1.Volume
		wantVolLen     int
		wantVolNames   []string // expected volume names in order
	}{
		{
			name:           "adds volume when missing",
			initialVolumes: []corev1.Volume{},
			wantVolLen:     1,
			wantVolNames:   []string{"kube-api-access-mca-sa"},
		},
		{
			name: "does not add duplicate volume",
			initialVolumes: []corev1.Volume{
				{
					Name: "kube-api-access-mca-sa",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			wantVolLen:   1,
			wantVolNames: []string{"kube-api-access-mca-sa"},
		},
		{
			name: "preserves existing volumes",
			initialVolumes: []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			wantVolLen:   3,
			wantVolNames: []string{"data", "config", "kube-api-access-mca-sa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: tt.initialVolumes,
				},
			}

			addRequiredVolume(pod)

			require.Len(t, pod.Spec.Volumes, tt.wantVolLen)
			for i, name := range tt.wantVolNames {
				assert.Equal(t, name, pod.Spec.Volumes[i].Name)
			}

			// Verify MCA volume has EmptyDir
			for _, vol := range pod.Spec.Volumes {
				if vol.Name == "kube-api-access-mca-sa" {
					assert.NotNil(t, vol.EmptyDir)
				}
			}
		})
	}
}

func TestInjectProxy_MultipleContainersWithMixedVolumeMounts(t *testing.T) {
	pod := corev1.Pod{
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
				{
					Name:  "sidecar",
					Image: "sidecar",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/data",
						},
					},
				},
				{
					Name:  "another-app",
					Image: "another",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kube-api-access-2",
							MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
						},
					},
				},
			},
		},
	}

	result, err := injectProxy(pod)
	require.NoError(t, err)

	require.Len(t, result.Spec.Containers, 3)

	// First container: existing mount updated
	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Containers[0].VolumeMounts[0].Name)
	assert.Len(t, result.Spec.Containers[0].Env, 2)

	// Second container: mount added (now has 2 mounts)
	assert.Len(t, result.Spec.Containers[1].VolumeMounts, 2)
	assert.Equal(t, "data", result.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Containers[1].VolumeMounts[1].Name)
	assert.Len(t, result.Spec.Containers[1].Env, 2)

	// Third container: existing mount updated
	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Containers[2].VolumeMounts[0].Name)
	assert.Len(t, result.Spec.Containers[2].Env, 2)
}
