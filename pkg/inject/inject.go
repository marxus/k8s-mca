// Package inject provides pod mutation logic for injecting the MCA proxy sidecar container.
// It modifies pod specifications to add the proxy container, configure volume mounts,
// and set environment variables to redirect Kubernetes API traffic through the local proxy.
package inject

import (
	"fmt"

	"github.com/marxus/k8s-mca/conf"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var proxyContainerYAML = `
name: mca-proxy
restartPolicy: Always
imagePullPolicy: Always # TODO: remove this in the end
securityContext: { runAsNonRoot: true, runAsUser: 999 }
args: [--proxy]
env:
  - name: NAMESPACE
    valueFrom: { fieldRef: { fieldPath: metadata.namespace } }
volumeMounts:
  - name: kube-api-access-mca-sa
    mountPath: /var/run/secrets/kubernetes.io/mca-serviceaccount
`

// ViaCLI injects the MCA proxy container into a pod from YAML input.
// It unmarshals the pod YAML, injects the proxy, and returns the mutated pod as YAML.
//
// Returns an error if unmarshaling fails, injection fails, or marshaling fails.
func ViaCLI(podYAML []byte) ([]byte, error) {
	var pod corev1.Pod
	if err := yaml.Unmarshal(podYAML, &pod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod: %w", err)
	}

	mutatedPod, err := injectProxy(pod)
	if err != nil {
		return nil, err
	}

	mutatedPodYAML, err := yaml.Marshal(&mutatedPod)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pod: %w", err)
	}

	return mutatedPodYAML, nil
}

// ViaWebhook injects the MCA proxy container into a pod from a webhook admission request.
// It injects the proxy sidecar and configures containers to use the local proxy endpoint.
//
// Returns the mutated pod and an error if injection fails.
func ViaWebhook(pod corev1.Pod) (corev1.Pod, error) {
	return injectProxy(pod)
}

func injectProxy(pod corev1.Pod) (corev1.Pod, error) {
	var proxyContainer corev1.Container
	var filteredInitContainers []corev1.Container
	for _, container := range pod.Spec.InitContainers {
		if container.Name == "mca-proxy" {
			proxyContainer = container
		} else {
			filteredInitContainers = append(filteredInitContainers, container)
		}
	}

	if proxyContainer.Image == "" {
		if err := yaml.Unmarshal([]byte(proxyContainerYAML), &proxyContainer); err != nil {
			return corev1.Pod{}, fmt.Errorf("failed to create MCA container: %w", err)
		}
		proxyContainer.Image = conf.ProxyImage
	}

	pod.Spec.InitContainers = append([]corev1.Container{proxyContainer}, filteredInitContainers...)

	for i := range filteredInitContainers {
		container := &filteredInitContainers[i]
		addVolumeMount(container)
		addEnvVars(container)
	}

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		addVolumeMount(container)
		addEnvVars(container)
	}

	addRequiredVolume(&pod)

	return pod, nil
}

func addVolumeMount(container *corev1.Container) {
	mount := corev1.VolumeMount{
		Name:      "kube-api-access-mca-sa",
		MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
		ReadOnly:  true,
	}

	for i := range container.VolumeMounts {
		if container.VolumeMounts[i].MountPath == mount.MountPath {
			container.VolumeMounts[i] = mount
			return
		}
	}
	container.VolumeMounts = append(container.VolumeMounts, mount)
}

func addEnvVars(container *corev1.Container) {
	envVars := map[string]string{
		"KUBERNETES_SERVICE_HOST": "127.0.0.1",
		"KUBERNETES_SERVICE_PORT": "6443",
	}

	for envName, envValue := range envVars {
		found := false
		for i := range container.Env {
			env := &container.Env[i]
			if env.Name == envName {
				env.Value = envValue
				found = true
				break
			}
		}
		if !found {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envName,
				Value: envValue,
			})
		}
	}
}

func addRequiredVolume(pod *corev1.Pod) {
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == "kube-api-access-mca-sa" {
			return
		}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name:         "kube-api-access-mca-sa",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	})
}
