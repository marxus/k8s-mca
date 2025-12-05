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

func TestViaCLI_ValidPod(t *testing.T) {
	podYAML := []byte(`
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
`)

	result, err := ViaCLI(podYAML)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	var resultPod corev1.Pod
	err = yaml.Unmarshal(result, &resultPod)
	require.NoError(t, err)

	assert.Len(t, resultPod.Spec.InitContainers, 1)
	assert.Equal(t, "mca-proxy", resultPod.Spec.InitContainers[0].Name)
}

func TestViaCLI_InvalidYAML(t *testing.T) {
	podYAML := []byte(`invalid yaml: {{{`)

	_, err := ViaCLI(podYAML)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal pod")
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

func TestInjectProxy_DoesNotUpdateContainerWithoutServiceAccountMount(t *testing.T) {
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

	assert.Equal(t, "data", container.VolumeMounts[0].Name)
	assert.Empty(t, container.Env)
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

func TestAddVolumeMount_UpdatesExistingMount(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "original-name",
				MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
			},
		},
	}

	result := addVolumeMount(container)

	assert.True(t, result)
	assert.Equal(t, "kube-api-access-mca-sa", container.VolumeMounts[0].Name)
	assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount", container.VolumeMounts[0].MountPath)
}

func TestAddVolumeMount_ReturnsFalseWhenNoMatch(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
			},
		},
	}

	result := addVolumeMount(container)

	assert.False(t, result)
	assert.Equal(t, "data", container.VolumeMounts[0].Name)
}

func TestAddVolumeMount_HandlesEmptyVolumeMounts(t *testing.T) {
	container := &corev1.Container{
		Name:         "app",
		VolumeMounts: []corev1.VolumeMount{},
	}

	result := addVolumeMount(container)

	assert.False(t, result)
	assert.Empty(t, container.VolumeMounts)
}

func TestAddEnvVars_AddsNewEnvVars(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env:  []corev1.EnvVar{},
	}

	addEnvVars(container)

	require.Len(t, container.Env, 2)

	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}

	assert.Equal(t, "127.0.0.1", envMap["KUBERNETES_SERVICE_HOST"])
	assert.Equal(t, "6443", envMap["KUBERNETES_SERVICE_PORT"])
}

func TestAddEnvVars_UpdatesExistingEnvVars(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "KUBERNETES_SERVICE_HOST", Value: "old-value"},
			{Name: "OTHER_VAR", Value: "keep-me"},
		},
	}

	addEnvVars(container)

	require.Len(t, container.Env, 3)

	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}

	assert.Equal(t, "127.0.0.1", envMap["KUBERNETES_SERVICE_HOST"])
	assert.Equal(t, "6443", envMap["KUBERNETES_SERVICE_PORT"])
	assert.Equal(t, "keep-me", envMap["OTHER_VAR"])
}

func TestAddEnvVars_PreservesOtherEnvVars(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "APP_ENV", Value: "production"},
			{Name: "DEBUG", Value: "false"},
		},
	}

	addEnvVars(container)

	require.Len(t, container.Env, 4)

	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}

	assert.Equal(t, "production", envMap["APP_ENV"])
	assert.Equal(t, "false", envMap["DEBUG"])
	assert.Equal(t, "127.0.0.1", envMap["KUBERNETES_SERVICE_HOST"])
	assert.Equal(t, "6443", envMap["KUBERNETES_SERVICE_PORT"])
}

func TestAddRequiredVolume_AddsVolumeWhenMissing(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{},
		},
	}

	addRequiredVolume(pod)

	require.Len(t, pod.Spec.Volumes, 1)
	assert.Equal(t, "kube-api-access-mca-sa", pod.Spec.Volumes[0].Name)
	assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
}

func TestAddRequiredVolume_DoesNotAddDuplicateVolume(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
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

	addRequiredVolume(pod)

	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Equal(t, "kube-api-access-mca-sa", pod.Spec.Volumes[0].Name)
}

func TestAddRequiredVolume_PreservesExistingVolumes(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
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
		},
	}

	addRequiredVolume(pod)

	require.Len(t, pod.Spec.Volumes, 3)
	assert.Equal(t, "data", pod.Spec.Volumes[0].Name)
	assert.Equal(t, "config", pod.Spec.Volumes[1].Name)
	assert.Equal(t, "kube-api-access-mca-sa", pod.Spec.Volumes[2].Name)
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

	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Containers[0].VolumeMounts[0].Name)
	assert.Len(t, result.Spec.Containers[0].Env, 2)

	assert.Equal(t, "data", result.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Empty(t, result.Spec.Containers[1].Env)

	assert.Equal(t, "kube-api-access-mca-sa", result.Spec.Containers[2].VolumeMounts[0].Name)
	assert.Len(t, result.Spec.Containers[2].Env, 2)
}
