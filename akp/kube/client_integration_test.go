//go:build kubeintegration && !acc

package kube

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// Run explicitly against a disposable/local cluster:
// AKP_KUBE_TEST_CONTEXT=orbstack go test -tags kubeintegration ./terraform/akp/kube -run TestClientIntegrationSuite -v
// Only ConfigMaps in one generated namespace are changed; no CRDs are deleted.
type ClientIntegrationSuite struct {
	suite.Suite
	client    *Client
	dyn       dynamic.Interface
	namespace string
	ctx       context.Context
}

func TestClientIntegrationSuite(t *testing.T) { suite.Run(t, new(ClientIntegrationSuite)) }
func (s *ClientIntegrationSuite) SetupSuite() {
	name := os.Getenv("AKP_KUBE_TEST_CONTEXT")
	s.Require().NotEmpty(name, "set AKP_KUBE_TEST_CONTEXT explicitly for this opt-in integration suite")
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{CurrentContext: name}).ClientConfig()
	s.Require().NoError(err)
	s.client, err = NewClient(cfg)
	s.Require().NoError(err)
	s.dyn = s.client.dyn
	var cancel context.CancelFunc
	s.ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	s.T().Cleanup(cancel)
	ns, err := s.dyn.Resource(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}).Create(s.ctx, &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"generateName": "tf-provider-regression-"}}}, metav1.CreateOptions{})
	s.Require().NoError(err)
	s.namespace = ns.GetName()
	s.T().Cleanup(func() {
		ctx, done := context.WithTimeout(context.Background(), time.Minute)
		defer done()
		// Release only the test's finalizer, even if an assertion failed.
		maps := s.dyn.Resource(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).Namespace(s.namespace)
		list, err := maps.List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, cm := range list.Items {
				_, err = maps.Patch(ctx, cm.GetName(), types.MergePatchType, []byte(`{"metadata":{"finalizers":[]}}`), metav1.PatchOptions{})
				s.NoError(err)
			}
		}
		ns := &unstructured.Unstructured{}
		ns.SetAPIVersion("v1")
		ns.SetKind("Namespace")
		ns.SetName(s.namespace)
		s.NoError(s.client.Delete(ctx, ns))
	})
}

func (s *ClientIntegrationSuite) object(name string, data map[string]any) *unstructured.Unstructured {
	cm := configMap(name)
	cm.SetNamespace(s.namespace)
	cm.Object["data"] = data
	return cm
}

func (s *ClientIntegrationSuite) maps() dynamic.ResourceInterface {
	return s.dyn.Resource(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).Namespace(s.namespace)
}

func (s *ClientIntegrationSuite) TestLegacyOwnershipPrunesOnlyProviderFields() {
	for i, manager := range []string{"HashiCorp", "kubectl-client-side-apply"} {
		s.Run(manager, func() {
			cm := s.object("legacy-"+string(rune('a'+i)), map[string]any{"desired": "old", "obsolete": "remove"})
			raw, err := json.Marshal(cm.Object)
			s.Require().NoError(err)
			cm.SetAnnotations(map[string]string{corev1.LastAppliedConfigAnnotation: string(raw)})
			live, err := s.maps().Create(s.ctx, cm, metav1.CreateOptions{FieldManager: manager})
			s.Require().NoError(err)
			uid := live.GetUID()
			foreign := s.object(cm.GetName(), map[string]any{"foreign": "keep"})
			raw, err = json.Marshal(foreign.Object)
			s.Require().NoError(err)
			_, err = s.maps().Patch(s.ctx, cm.GetName(), types.ApplyPatchType, raw, metav1.PatchOptions{FieldManager: "another-manager"})
			s.Require().NoError(err)
			desired := s.object(cm.GetName(), map[string]any{"desired": "new"})
			s.Require().NoError(s.client.Apply(s.ctx, desired))
			live, err = s.maps().Get(s.ctx, cm.GetName(), metav1.GetOptions{})
			s.Require().NoError(err)
			s.Equal(uid, live.GetUID())
			s.Equal(map[string]any{"desired": "new", "foreign": "keep"}, live.Object["data"])
			s.NotContains(live.GetAnnotations(), corev1.LastAppliedConfigAnnotation)
			s.Require().NoError(s.client.Delete(s.ctx, desired))
		})
	}
}

func (s *ClientIntegrationSuite) TestForceOwnershipAndLaterPruning() {
	cm := s.object("foreign-owner", map[string]any{"shared": "provider", "obsolete": "remove"})
	s.Require().NoError(s.client.Apply(s.ctx, cm))
	foreign := s.object(cm.GetName(), map[string]any{"shared": "foreign", "unrelated": "keep"})
	raw, err := json.Marshal(foreign.Object)
	s.Require().NoError(err)
	_, err = s.maps().Patch(s.ctx, cm.GetName(), types.ApplyPatchType, raw, metav1.PatchOptions{FieldManager: "another-manager", Force: new(true)})
	s.Require().NoError(err)
	cm.Object["data"] = map[string]any{"shared": "provider"}
	s.Require().NoError(s.client.Apply(s.ctx, cm))
	live, err := s.maps().Get(s.ctx, cm.GetName(), metav1.GetOptions{})
	s.Require().NoError(err)
	s.Equal(map[string]any{"shared": "provider", "unrelated": "keep"}, live.Object["data"])
	cm.Object["data"] = map[string]any{}
	s.Require().NoError(s.client.Apply(s.ctx, cm))
	live, err = s.maps().Get(s.ctx, cm.GetName(), metav1.GetOptions{})
	s.Require().NoError(err)
	s.Equal(map[string]any{"unrelated": "keep"}, live.Object["data"])
	s.Require().NoError(s.client.Delete(s.ctx, cm))
}

func (s *ClientIntegrationSuite) TestFinalizerDeadlineThenRetry() {
	cm := s.object("held", map[string]any{"key": "value"})
	cm.SetFinalizers([]string{"test.akuity.io/hold"})
	_, err := s.maps().Create(s.ctx, cm, metav1.CreateOptions{})
	s.Require().NoError(err)
	ctx, cancel := context.WithTimeout(s.ctx, 250*time.Millisecond)
	defer cancel()
	s.ErrorIs(s.client.Delete(ctx, cm), context.DeadlineExceeded)
	live, err := s.maps().Get(s.ctx, cm.GetName(), metav1.GetOptions{})
	s.Require().NoError(err)
	s.Require().NotNil(live.GetDeletionTimestamp())
	// Retry must still wait for actual removal, not merely accept the DELETE response.
	done := make(chan error, 1)
	go func() { done <- s.client.Delete(s.ctx, cm) }()
	select {
	case err := <-done:
		s.FailNow("delete completed while finalizer still exists", "%v", err)
	case <-time.After(100 * time.Millisecond):
	}

	_, err = s.maps().Patch(s.ctx, cm.GetName(), types.MergePatchType, []byte(`{"metadata":{"finalizers":[]}}`), metav1.PatchOptions{})
	s.Require().NoError(err)
	select {
	case err := <-done:
		s.NoError(err)
	case <-s.ctx.Done():
		s.FailNow("delete did not finish after removing finalizer")
	}
	s.NoError(s.client.Delete(s.ctx, cm))
}
