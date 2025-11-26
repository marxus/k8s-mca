package inject

import (
	"fmt"

	"github.com/lithammer/dedent"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var mcaContainerYAML = dedent.Dedent(`
	name: mca
	image: mca:latest
	restartPolicy: Always
	volumeMounts:
	  - name: kube-api-access-sa
		mountPath: /var/run/secrets/kubernetes.io/serviceaccount
		readOnly: true
	  - name: kube-api-access-mca-sa
		mountPath: /var/run/secrets/kubernetes.io/mca-serviceaccount
`)

var requiredVolumesYAML = dedent.Dedent(`
	- name: kube-api-access-sa
	  projected:
		sources:
		  - serviceAccountToken:
			  path: token
			  expirationSeconds: 3607
		  - configMap:
			  name: kube-root-ca.crt
			  items:
				- key: ca.crt
				  path: ca.crt
		  - downwardAPI:
			  items:
				- path: namespace
				  fieldRef:
					fieldPath: metadata.namespace
	- name: kube-api-access-mca-sa
	  emptyDir: {}
`)

func InjectMCA(podYAML []byte) ([]byte, error) {
	var pod corev1.Pod
	if err := yaml.Unmarshal(podYAML, &pod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod: %w", err)
	}

	// Set automountServiceAccountToken to false
	automount := false
	pod.Spec.AutomountServiceAccountToken = &automount

	// Remove any existing MCA init containers
	var filteredInitContainers []corev1.Container
	for _, container := range pod.Spec.InitContainers {
		if container.Name != "mca" {
			filteredInitContainers = append(filteredInitContainers, container)
		}
	}

	// Create MCA init container
	// TODO: need to decide: should check and respect explicit user configuration for automountServiceAccountToken inorder to decide if MCA should have the `kube-api-access-sa` volume or not?
	var mcaContainer corev1.Container
	if err := yaml.Unmarshal([]byte(mcaContainerYAML), &mcaContainer); err != nil {
		return nil, fmt.Errorf("failed to create MCA container: %w", err)
	}

	// Prepend MCA as first init container
	pod.Spec.InitContainers = append([]corev1.Container{mcaContainer}, filteredInitContainers...)

	// Add environment variables to all non-MCA init containers
	for i := range filteredInitContainers {
		addEnvVars(&filteredInitContainers[i])
		addVolumeMount(&filteredInitContainers[i])
	}

	// Add environment variables to all regular containers
	for i := range pod.Spec.Containers {
		addEnvVars(&pod.Spec.Containers[i])
		addVolumeMount(&pod.Spec.Containers[i])
	}

	// Ensure required volumes exist
	if err := addRequiredVolumes(&pod); err != nil {
		return nil, err
	}

	// Marshal back to YAML
	result, err := yaml.Marshal(&pod)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pod: %w", err)
	}

	return result, nil
}

func addEnvVars(container *corev1.Container) {
	// Check if env vars already exist and update or add
	envVars := map[string]string{
		"KUBERNETES_SERVICE_HOST": "127.0.0.1",
		"KUBERNETES_SERVICE_PORT": "6443",
	}

	for envName, envValue := range envVars {
		found := false
		for i := range container.Env {
			if container.Env[i].Name == envName {
				container.Env[i].Value = envValue
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

func addVolumeMount(container *corev1.Container) {
	mountName := "kube-api-access-mca-sa"
	mountPath := "/var/run/secrets/kubernetes.io/serviceaccount"

	// Check if mount already exists
	for i := range container.VolumeMounts {
		if container.VolumeMounts[i].Name == mountName {
			container.VolumeMounts[i].MountPath = mountPath
			container.VolumeMounts[i].ReadOnly = true
			return
		}
	}

	// Add new mount
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      mountName,
		MountPath: mountPath,
		ReadOnly:  true,
	})
}

func addRequiredVolumes(pod *corev1.Pod) error {
	// Remove any existing required volumes to avoid duplicates
	var filteredVolumes []corev1.Volume
	for _, vol := range pod.Spec.Volumes {
		if vol.Name != "kube-api-access-sa" && vol.Name != "kube-api-access-mca-sa" {
			filteredVolumes = append(filteredVolumes, vol)
		}
	}

	// Add required volumes
	var requiredVolumes []corev1.Volume
	if err := yaml.Unmarshal([]byte(requiredVolumesYAML), &requiredVolumes); err != nil {
		return fmt.Errorf("failed to create required volumes: %w", err)
	}
	filteredVolumes = append(filteredVolumes, requiredVolumes...)

	// Replace volumes with filtered + required volumes
	pod.Spec.Volumes = filteredVolumes

	return nil
}
