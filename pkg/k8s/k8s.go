package k8s

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/srivickynesh/release-tests-ginkgopkg/clients"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/config"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/oc"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/openshift"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/store"
	secv1 "github.com/openshift/api/security/v1"
	secclient "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	"github.com/tektoncd/pipeline/pkg/names"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

// ==============================
// 1. ClientSet & Deployment Utils
// ==============================

func NewClientSet() (*clients.Clients, string, func(), error) {
	ns := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("releasetest")
	cs, err := clients.NewClients(config.Flags.Kubeconfig, config.Flags.Cluster, ns)
	if err != nil {
		return nil, "", nil, err
	}
	oc.CreateNewProject(ns)
	return cs, ns, func() {
		oc.DeleteProjectIgnoreErors(ns)
	}, nil
}

func WaitForDeploymentDeletion(cs *clients.Clients, namespace, name string) error {
	return wait.PollUntilContextTimeout(cs.Ctx, config.APIRetry, config.APITimeout, false, func(context.Context) (bool, error) {
		kc := cs.KubeClient.Kube
		_, err := kc.AppsV1().Deployments(namespace).Get(cs.Ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsGone(err) || errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		log.Printf("Waiting for deletion of %s deployment\n", name)
		return false, nil
	})
}

func WaitForDeployment(ctx context.Context, kc kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, retryInterval, timeout, false, func(context.Context) (bool, error) {
		deployment, err := kc.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Waiting for availability of %s deployment\n", name)
				return false, nil
			}
			return false, err
		}
		if int(deployment.Status.AvailableReplicas) == replicas && int(deployment.Status.UnavailableReplicas) == 0 {
			return true, nil
		}
		log.Printf("Waiting for full availability of deployment %s (%d/%d)\n", name, deployment.Status.AvailableReplicas, replicas)
		return false, nil
	})
}

func ValidateDeployments(cs *clients.Clients, ns string, deployments ...string) error {
	kc := cs.KubeClient.Kube
	for _, d := range deployments {
		err := WaitForDeployment(cs.Ctx, kc, ns, d, 1, config.APIRetry, config.APITimeout)
		if err != nil {
			return fmt.Errorf("failed to create deployment %s: %w", d, err)
		}
	}
	return nil
}

func DeleteDeployment(cs *clients.Clients, ns string, deploymentName string) error {
	kc := cs.KubeClient.Kube
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := kc.AppsV1().Deployments(ns).Delete(ctx, deploymentName, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete deployment %s in namespace %s: %v", deploymentName, ns, err)
	}
	return WaitForDeploymentDeletion(cs, ns, deploymentName)
}

func ValidateDeploymentDeletion(cs *clients.Clients, ns string, deployments ...string) error {
	for _, d := range deployments {
		err := WaitForDeploymentDeletion(cs, ns, d)
		if err != nil {
			return fmt.Errorf("failed to delete deployment %s: %w", d, err)
		}
	}
	return nil
}

// ==============================
// 2. ServiceAccount Utilities
// ==============================

func WaitForServiceAccount(cs *clients.Clients, ns, targetSA string) (*corev1.ServiceAccount, error) {
	var ret *corev1.ServiceAccount
	err := wait.PollUntilContextTimeout(cs.Ctx, config.APIRetry, config.APITimeout, false, func(context.Context) (bool, error) {
		saList, err := cs.KubeClient.Kube.CoreV1().ServiceAccounts(ns).List(cs.Ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, sa := range saList.Items {
			if sa.Name == targetSA {
				saCopy := sa
				ret = &saCopy
				return true, nil
			}
		}
		return false, nil
	})
	return ret, err
}

func VerifyNoServiceAccount(ctx context.Context, kc *clients.KubeClient, sa, ns string) error {
	return wait.PollUntilContextTimeout(ctx, config.APIRetry, config.APITimeout, true, func(context.Context) (bool, error) {
		_, err := kc.Kube.CoreV1().ServiceAccounts(ns).Get(ctx, sa, metav1.GetOptions{})
		if err == nil || !errors.IsNotFound(err) {
			return false, fmt.Errorf("service account %q still exists in namespace %q", sa, ns)
		}
		return true, nil
	})
}

func VerifyServiceAccountExists(ctx context.Context, kc *clients.KubeClient, sa, ns string) error {
	return wait.PollUntilContextTimeout(ctx, config.APIRetry, config.APITimeout, true, func(context.Context) (bool, error) {
		_, err := kc.Kube.CoreV1().ServiceAccounts(ns).Get(ctx, sa, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func VerifyNamespaceExists(ctx context.Context, kc *clients.KubeClient, ns string) error {
	return wait.PollUntilContextTimeout(ctx, config.APIRetry, config.APITimeout, true, func(context.Context) (bool, error) {
		_, err := kc.Kube.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// ==============================
// 3. CronJob Utilities
// ==============================

func CreateCronJob(c *clients.Clients, args []string, schedule, namespace string) (string, error) {
	cronjob := &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{APIVersion: batchv1.SchemeGroupVersion.String(), Kind: "CronJob"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: "hello"},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "hello", Image: "image-registry.openshift-image-registry.svc:5000/openshift/golang", Args: args}},
							RestartPolicy: corev1.RestartPolicy("Never"),
						},
					},
				},
			},
		},
	}
	cj, err := c.KubeClient.Kube.BatchV1().CronJobs(namespace).Create(c.Ctx, cronjob, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	store.PutScenarioData("cronjob", cj.Name)
	return cj.Name, nil
}

func DeleteCronJob(c *clients.Clients, name, namespace string) error {
	policy := metav1.DeletePropagationBackground
	return c.KubeClient.Kube.BatchV1().CronJobs(namespace).Delete(c.Ctx, name, metav1.DeleteOptions{PropagationPolicy: &policy})
}

func AssertIfDefaultCronjobExists(c *clients.Clients, namespace string) error {
	cronJobs, err := c.KubeClient.Kube.BatchV1().CronJobs(namespace).List(c.Ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list cronjobs in namespace %s: %w", namespace, err)
	}
	if len(cronJobs.Items) == 0 {
		return fmt.Errorf("no cronjobs present in namespace %s", namespace)
	}
	for _, cj := range cronJobs.Items {
		if cj.Spec.Schedule == config.PrunerSchedule && strings.Contains(cj.Name, config.PrunerNamePrefix) {
			return nil
		}
	}
	return fmt.Errorf("no cronjob with schedule %v and prefix %v present", config.PrunerSchedule, config.PrunerNamePrefix)
}

func GetCronjobNameWithSchedule(c *clients.Clients, namespace, schedule string) (string, error) {
	cronJobs, err := c.KubeClient.Kube.BatchV1().CronJobs(namespace).List(c.Ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, cj := range cronJobs.Items {
		if cj.Spec.Schedule == schedule && strings.Contains(cj.Name, "tekton-resource-pruner-") {
			return cj.Name, nil
		}
	}
	return "", fmt.Errorf("no cronjob with schedule %s found", schedule)
}

func AssertPrunerCronjobWithContainer(c *clients.Clients, namespace string, num int) error {
	cronJobs, err := c.KubeClient.Kube.BatchV1().CronJobs(namespace).List(c.Ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, cj := range cronJobs.Items {
		if strings.Contains(cj.Name, "tekton-resource-pruner") {
			if len(cj.Spec.JobTemplate.Spec.Template.Spec.Containers) != num {
				return fmt.Errorf("expected %d containers in cronjob %s but found %d", num, cj.Name, len(cj.Spec.JobTemplate.Spec.Template.Spec.Containers))
			}
			return nil
		}
	}
	return fmt.Errorf("cronjob with prefix tekton-resource-pruner not found in %s", namespace)
}

func AssertCronjobPresent(c *clients.Clients, cronJobName, namespace string) error {
	return wait.PollUntilContextTimeout(c.Ctx, config.APIRetry, config.ResourceTimeout, false, func(context.Context) (bool, error) {
		cronJobs, err := c.KubeClient.Kube.BatchV1().CronJobs(namespace).List(c.Ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, cj := range cronJobs.Items {
			if strings.Contains(cj.Name, cronJobName) {
				return true, nil
			}
		}
		return false, nil
	})
}

func AssertCronjobNotPresent(c *clients.Clients, cronJobName, namespace string) error {
}

// ==============================
// 4. Tekton InstallerSet & Dynamic Resource Utilities
// ==============================

// ValidateTektonInstallersetStatus checks that all TektonInstallerSets are in Ready state.
func ValidateTektonInstallersetStatus(c *clients.Clients) error {
	tis, err := c.Operator.TektonInstallerSets().List(c.Ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error getting tektoninstallersets: %w", err)
	}
	var failed []string
	for _, is := range tis.Items {
		if !is.Status.IsReady() {
			failed = append(failed, is.Name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("installer sets not ready: %s", strings.Join(failed, ","))
	}
	return nil
}

// ValidateTektonInstallersetNames verifies that installer sets with configured prefixes exist.
func ValidateTektonInstallersetNames(c *clients.Clients) error {
	tis, err := c.Operator.TektonInstallerSets().List(c.Ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error getting tektoninstallersets: %w", err)
	}
	var missing []string
	for _, prefix := range config.TektonInstallersetNamePrefixes {
		if !openshift.IsCapabilityEnabled(c, "Console") &&
			(prefix == "addon-custom-consolecli" || prefix == "addon-custom-openshiftconsole") {
			continue
		}
		if config.Flags.IsDisconnected && prefix == "addon-custom-communityclustertask" {
			continue
		}
		found := false
		for _, is := range tis.Items {
			if strings.HasPrefix(is.Name, prefix) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, prefix)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("installer sets missing for prefixes: %s", strings.Join(missing, ","))
	}
	return nil
}

// GetWarningEvents retrieves all Warning events in a namespace.
func GetWarningEvents(c *clients.Clients, namespace string) (string, error) {
	var eventsMsgs []string
	events, err := c.KubeClient.Kube.CoreV1().Events(namespace).List(c.Ctx, metav1.ListOptions{FieldSelector: "type=Warning"})
	if err != nil {
		return "", err
	}
	for _, e := range events.Items {
		eventsMsgs = append(eventsMsgs, e.Message)
	}
	return strings.Join(eventsMsgs, "
"), nil
}

// Get retrieves a dynamic resource by GroupVersionResource.
func Get(ctx context.Context, gr schema.GroupVersionResource, clients *clients.Clients, name, namespace string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	gvr, err := GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}
	return clients.Dynamic.Resource(*gvr).Namespace(namespace).Get(ctx, name, opts)
}

// Watch watches a dynamic resource and returns a watch.Interface.
func Watch(ctx context.Context, gr schema.GroupVersionResource, clients *clients.Clients, namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	gvr, err := GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}
	return clients.Dynamic.Resource(*gvr).Namespace(namespace).Watch(ctx, opts)
}

// GetGroupVersionResource maps a GroupVersionResource to a REST mapping using discovery.
func GetGroupVersionResource(gr schema.GroupVersionResource, discovery discovery.DiscoveryInterface) (*schema.GroupVersionResource, error) {
	apiRes, err := restmapper.GetAPIGroupResources(discovery)
	if err != nil {
		return nil, err
	}
	rm := restmapper.NewDiscoveryRESTMapper(apiRes)
	gvr, err := rm.ResourceFor(gr)
	if err != nil {
		return nil, err
	}
	return &gvr, nil
}
