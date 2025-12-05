package inject

import (
	"fmt"

	"github.com/marxus/k8s-mca/conf"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var proxyContainerYAML = fmt.Sprintf(`
name: mca-proxy
restartPolicy: Always
imagePullPolicy: Always # TODO: remove this in the end
securityContext: { runAsNonRoot: true, runAsUser: 999 }
args: [--proxy]
volumeMounts:
  - name: kube-api-access-mca-sa
    mountPath: %s
`, conf.MCAServiceAccountPath)

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
		container := &pod.Spec.Containers[i]
		if addVolumeMount(container) {
			addEnvVars(container)
		}
	}

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if addVolumeMount(container) {
			addEnvVars(container)
		}
	}

	addRequiredVolume(&pod)

	return pod, nil
}

func addVolumeMount(container *corev1.Container) bool {
	for i := range container.VolumeMounts {
		mount := &container.VolumeMounts[i]
		if mount.MountPath == conf.ServiceAccountPath {
			mount.Name = "kube-api-access-mca-sa"
			return true
		}
	}
	return false
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
