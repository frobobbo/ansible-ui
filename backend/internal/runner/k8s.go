package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sRunner executes ansible-playbook inside a Kubernetes Job using a
// container image (Execution Environment) instead of an SSH connection.
type K8sRunner struct {
	client    kubernetes.Interface
	namespace string
}

var globalK8sRunner *K8sRunner

// GetK8sRunner returns a shared K8sRunner, initializing it on first call.
// Returns an error if the Kubernetes API is unreachable.
func GetK8sRunner() (*K8sRunner, error) {
	if globalK8sRunner != nil {
		return globalK8sRunner, nil
	}
	r, err := newK8sRunner()
	if err != nil {
		return nil, err
	}
	globalK8sRunner = r
	return globalK8sRunner, nil
}

func newK8sRunner() (*K8sRunner, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to KUBECONFIG for local development.
		kc := os.Getenv("KUBECONFIG")
		if kc == "" {
			if home := os.Getenv("HOME"); home != "" {
				kc = home + "/.kube/config"
			}
		}
		cfg, err = clientcmd.BuildConfigFromFlags("", kc)
		if err != nil {
			return nil, fmt.Errorf("k8s config: %w", err)
		}
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s client: %w", err)
	}
	return &K8sRunner{client: client, namespace: k8sNamespace()}, nil
}

func k8sNamespace() string {
	if ns := os.Getenv("K8S_NAMESPACE"); ns != "" {
		return ns
	}
	// When running inside a pod the namespace is injected here by Kubernetes.
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return "default"
}

// RunPlaybook creates a Kubernetes Job that runs ansible-playbook inside the
// given container image, streams its output to outputCh, and returns the result.
// All temporary resources (Job, ConfigMap, Secret) are cleaned up on return.
func (r *K8sRunner) RunPlaybook(
	ctx context.Context,
	runID string,
	image string,
	playbookContent []byte,
	inventoryTarget string,
	variables map[string]interface{},
	preCommand string,
	vaultPassword string,
	vaultFileContent []byte,
	vaultFileName string,
	sshCertContent []byte,
	outputCh chan<- string,
) RunResult {
	// Resource names are derived from the first 8 chars of the run UUID.
	prefix := "af-" + strings.ReplaceAll(runID, "-", "")[:8]
	labels := map[string]string{"ansible-frontend/run-id": runID}

	// If an SSH cert is provided, inject the key path into inventory before mounting.
	// We copy the Secret-mounted file to /tmp/ansible-key and chmod 600 it in the
	// shell command because Secret volumes are root-owned; a non-root UID can only
	// read them if we set defaultMode 0644, but SSH rejects keys that aren't 0600
	// owned by the current user. Copying to /tmp gives us a user-owned 0600 copy.
	if len(sshCertContent) > 0 && inventoryTarget != "" {
		inventoryTarget = strings.TrimSuffix(inventoryTarget, "\n") + " ansible_ssh_private_key_file=/tmp/ansible-key\n"
	}

	// ── ConfigMap: playbook YAML (+ inventory + vault vars file if any) ────────
	cmData := map[string]string{"playbook.yml": string(playbookContent)}
	if inventoryTarget != "" {
		cmData["inventory"] = inventoryTarget
	}
	if len(vaultFileContent) > 0 {
		cmData["vault-vars.yml"] = string(vaultFileContent)
		// Also store under the original filename so the SubPath mount can place it
		// at /ansible/<stem>/<filename> for vars_files: ./creds/creds.yml lookups.
		if vaultFileName != "" {
			cmData[vaultFileName] = string(vaultFileContent)
		}
	}
	cm, err := r.client.CoreV1().ConfigMaps(r.namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: prefix + "-cm", Namespace: r.namespace, Labels: labels},
		Data:       cmData,
	}, metav1.CreateOptions{})
	if err != nil {
		return RunResult{Err: fmt.Errorf("create configmap: %w", err)}
	}
	defer r.client.CoreV1().ConfigMaps(r.namespace).Delete(
		context.Background(), cm.Name, metav1.DeleteOptions{})

	// ── Secret: vault password (only when a vault is attached) ───────────────
	var secretName string
	if vaultPassword != "" {
		secret, serr := r.client.CoreV1().Secrets(r.namespace).Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: prefix + "-vault", Namespace: r.namespace, Labels: labels},
			StringData: map[string]string{"password": vaultPassword},
		}, metav1.CreateOptions{})
		if serr != nil {
			return RunResult{Err: fmt.Errorf("create vault secret: %w", serr)}
		}
		secretName = secret.Name
		defer r.client.CoreV1().Secrets(r.namespace).Delete(
			context.Background(), secretName, metav1.DeleteOptions{})
	}

	// ── Secret: SSH private key (only when a host cert is attached) ──────────
	var certSecretName string
	if len(sshCertContent) > 0 {
		certSecret, cerr := r.client.CoreV1().Secrets(r.namespace).Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: prefix + "-cert", Namespace: r.namespace, Labels: labels},
			Data:       map[string][]byte{"key": sshCertContent},
		}, metav1.CreateOptions{})
		if cerr != nil {
			return RunResult{Err: fmt.Errorf("create cert secret: %w", cerr)}
		}
		certSecretName = certSecret.Name
		defer r.client.CoreV1().Secrets(r.namespace).Delete(
			context.Background(), certSecretName, metav1.DeleteOptions{})
	}

	// ── Build the shell command ───────────────────────────────────────────────
	varJSON, err := json.Marshal(variables)
	if err != nil {
		return RunResult{Err: fmt.Errorf("marshal vars: %w", err)}
	}
	varStr := strings.ReplaceAll(string(varJSON), "'", `'"'"'`)
	ansibleCmd := fmt.Sprintf("ansible-playbook /ansible/playbook.yml --extra-vars '%s'", varStr)
	if inventoryTarget != "" {
		ansibleCmd = fmt.Sprintf("ansible-playbook /ansible/playbook.yml -i /ansible/inventory --extra-vars '%s'", varStr)
	}
	if vaultPassword != "" {
		ansibleCmd += " --vault-password-file /ansible-vault/password"
	}
	if len(vaultFileContent) > 0 {
		ansibleCmd += " --extra-vars '@/ansible/vault-vars.yml'"
	}
	// EE containers often run as a non-root UID that has no /etc/passwd entry.
	// SSH requires the current UID to resolve to a username; if it can't, it
	// aborts with "No user exists for uid <N>". Prepend the standard OpenShift
	// arbitrary-UID fix: write a passwd entry only when one is missing.
	passwdFix := `if ! whoami &>/dev/null && [ -w /etc/passwd ]; then echo "user:x:$(id -u):$(id -g)::/tmp:/bin/sh" >> /etc/passwd; fi`
	preamble := passwdFix
	if len(sshCertContent) > 0 {
		preamble += " && cp /ansible-cert/key /tmp/ansible-key && chmod 600 /tmp/ansible-key"
	}

	shellCmd := preamble + " && " + ansibleCmd
	if preCommand != "" {
		shellCmd = preamble + " && " + preCommand + " && " + ansibleCmd
	}

	// ── Volumes & mounts ─────────────────────────────────────────────────────
	volumes := []corev1.Volume{{
		Name: "playbook",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
			},
		},
	}}
	mounts := []corev1.VolumeMount{{Name: "playbook", MountPath: "/ansible"}}

	// SubPath mount places the vault file at /ansible/<stem>/<filename> so
	// vars_files: ./creds/creds.yml resolves correctly without needing to write
	// to the read-only ConfigMap mount.
	if len(vaultFileContent) > 0 && vaultFileName != "" {
		stem := strings.TrimSuffix(vaultFileName, filepath.Ext(vaultFileName))
		if stem != "" && stem != vaultFileName {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      "playbook",
				MountPath: fmt.Sprintf("/ansible/%s/%s", stem, vaultFileName),
				SubPath:   vaultFileName,
			})
		}
	}

	if secretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "vault",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: secretName},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name: "vault", MountPath: "/ansible-vault", ReadOnly: true,
		})
	}

	if certSecretName != "" {
		var mode int32 = 0644 // readable by non-root; shell cmd copies to /tmp and chmod 600
		volumes = append(volumes, corev1.Volume{
			Name: "cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  certSecretName,
					DefaultMode: &mode,
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name: "cert", MountPath: "/ansible-cert", ReadOnly: true,
		})
	}

	// ── Create Job ───────────────────────────────────────────────────────────
	var backoffLimit int32 = 0
	var ttl int32 = 120 // auto-cleanup 2 min after completion
	job, err := r.client.BatchV1().Jobs(r.namespace).Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: prefix, Namespace: r.namespace, Labels: labels},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:         "ansible",
						Image:        image,
						Command:      []string{"sh", "-c", shellCmd},
						Env: []corev1.EnvVar{
							{Name: "ANSIBLE_FORCE_COLOR", Value: "1"},
							{Name: "HOME", Value: "/tmp"},
							{Name: "ANSIBLE_HOST_KEY_CHECKING", Value: "False"},
						},
						VolumeMounts: mounts,
					}},
					Volumes: volumes,
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return RunResult{Err: fmt.Errorf("create job: %w", err)}
	}
	defer func() {
		prop := metav1.DeletePropagationForeground
		r.client.BatchV1().Jobs(r.namespace).Delete(
			context.Background(), job.Name, metav1.DeleteOptions{PropagationPolicy: &prop})
	}()

	// ── Wait for pod to become running or terminal ────────────────────────────
	podName, err := r.waitForPod(ctx, runID, outputCh)
	if err != nil {
		return RunResult{Err: err}
	}

	// ── Stream logs ───────────────────────────────────────────────────────────
	exitCode := r.streamAndWait(ctx, podName, outputCh)
	return RunResult{ExitCode: exitCode}
}

// waitForPod polls until the Job's pod is Running, Succeeded, or Failed.
// It reports image-pull errors immediately so the user sees them in the log.
func (r *K8sRunner) waitForPod(ctx context.Context, runID string, outputCh chan<- string) (string, error) {
	sel := "ansible-frontend/run-id=" + runID
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("cancelled while waiting for pod to start")
		default:
		}

		pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{LabelSelector: sel})
		if err != nil {
			return "", fmt.Errorf("list pods: %w", err)
		}

		for _, pod := range pods.Items {
			// Detect image-pull failures early.
			for _, cs := range pod.Status.ContainerStatuses {
				if w := cs.State.Waiting; w != nil {
					if w.Reason == "ErrImagePull" || w.Reason == "ImagePullBackOff" {
						return "", fmt.Errorf("image pull failed (%s): %s", w.Reason, w.Message)
					}
				}
			}
			switch pod.Status.Phase {
			case corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed:
				return pod.Name, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("cancelled while waiting for pod to start")
		case <-time.After(2 * time.Second):
		}
	}
}

// streamAndWait streams the pod's log output to outputCh, waits for the
// container to terminate, and returns its exit code.
func (r *K8sRunner) streamAndWait(ctx context.Context, podName string, outputCh chan<- string) int {
	req := r.client.CoreV1().Pods(r.namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "ansible",
		Follow:    true,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		select {
		case outputCh <- fmt.Sprintf("error streaming logs: %v", err):
		default:
		}
		return 1
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case outputCh <- line:
		case <-ctx.Done():
			return 1
		}
	}

	// Retrieve exit code from the terminated container state.
	pod, err := r.client.CoreV1().Pods(r.namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return 1
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == "ansible" && cs.State.Terminated != nil {
			return int(cs.State.Terminated.ExitCode)
		}
	}
	return 0
}
