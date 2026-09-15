//go:build !acc

package kube

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/cached/memory"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/restmapper"
	clienttesting "k8s.io/client-go/testing"
)

func newTestClient(t *testing.T, objs ...runtime.Object) (*Client, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	disco := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{}}
	disco.Resources = []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{{Name: "configmaps", Kind: "ConfigMap", Namespaced: true}},
	}}
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
	return &Client{dyn: dyn, mapper: restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))}, dyn
}

func configMap(name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("ConfigMap")
	u.SetNamespace("akuity")
	u.SetName(name)
	return u
}

// blockedDelete returns a client whose DELETE is accepted while a finalizer keeps the object alive.
func blockedDelete(t *testing.T) (*Client, *dynamicfake.FakeDynamicClient, *unstructured.Unstructured) {
	t.Helper()
	cm := configMap("finalized")
	cm.SetFinalizers([]string{"test.akuity.io/hold"})
	client, dyn := newTestClient(t, cm)
	dyn.PrependReactor("delete", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) { return true, nil, nil })
	return client, dyn, cm
}

func TestClientDelete(t *testing.T) {
	ctx := context.Background()
	cm := configMap("agent")

	t.Run("missing object is not an error", func(t *testing.T) {
		client, _ := newTestClient(t)
		require.NoError(t, client.Delete(ctx, cm))
	})

	t.Run("waits until the object is gone", func(t *testing.T) {
		client, dyn := newTestClient(t, cm)
		require.NoError(t, client.Delete(ctx, cm))
		_, err := dyn.Resource(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).Namespace("akuity").Get(ctx, "agent", metav1.GetOptions{})
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("polling errors other than NotFound surface", func(t *testing.T) {
		client, dyn := newTestClient(t, cm)
		dyn.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "agent", nil)
		})
		err := client.Delete(ctx, cm)
		require.True(t, apierrors.IsForbidden(err), "got %v", err)
	})

	t.Run("waits for a finalizer and can be retried", func(t *testing.T) {
		client, dyn, cm := blockedDelete(t)
		polls := 0
		dyn.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
			polls++
			if polls == 1 {
				return true, cm, nil
			}
			return true, nil, apierrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, cm.GetName())
		})
		require.NoError(t, client.Delete(ctx, cm))
		require.Equal(t, 2, polls)
		require.NoError(t, client.Delete(ctx, cm))
	})

	t.Run("cancellation stops a blocked deletion", func(t *testing.T) {
		client, dyn, cm := blockedDelete(t)
		ctx, cancel := context.WithCancel(ctx)
		dyn.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
			cancel()
			return true, cm, nil
		})
		require.ErrorIs(t, client.Delete(ctx, cm), context.Canceled)
	})

	t.Run("caller deadline stops a blocked deletion", func(t *testing.T) {
		client, _, cm := blockedDelete(t)
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		require.ErrorIs(t, client.Delete(ctx, cm), context.DeadlineExceeded)
	})

	t.Run("no caller deadline still times out", func(t *testing.T) {
		client, _, cm := blockedDelete(t)
		restore := deleteTimeout
		deleteTimeout = 50 * time.Millisecond
		t.Cleanup(func() { deleteTimeout = restore })
		require.ErrorIs(t, client.Delete(ctx, cm), context.DeadlineExceeded)
	})

	t.Run("missing kind is reported and rediscovered on retry", func(t *testing.T) {
		client, dyn := newTestClient(t, cm)
		disco := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{}}
		disco.Resources = []*metav1.APIResourceList{{GroupVersion: "v1", APIResources: []metav1.APIResource{{Name: "secrets", Kind: "Secret", Namespaced: true}}}}
		client.mapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
		require.True(t, meta.IsNoMatchError(client.Delete(ctx, cm)))
		require.Empty(t, dyn.Actions(), "unresolvable kinds must not delete another resource")
		disco.Resources[0].APIResources = append(disco.Resources[0].APIResources, metav1.APIResource{Name: "configmaps", Kind: "ConfigMap", Namespaced: true})
		require.NoError(t, client.Delete(ctx, cm), "retry must invalidate stale discovery after the kind is restored")
	})
}

func TestClientApply(t *testing.T) {
	ctx := context.Background()
	cm := configMap("agent")

	// run applies cm against canned server responses and returns the patch types sent.
	run := func(t *testing.T, live *unstructured.Unstructured, conflicts int) ([]types.PatchType, error) {
		client, dyn := newTestClient(t)
		var sent []types.PatchType
		dyn.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, live, nil
		})
		dyn.PrependReactor("patch", "configmaps", func(a clienttesting.Action) (bool, runtime.Object, error) {
			p := a.(clienttesting.PatchAction)
			sent = append(sent, p.GetPatchType())
			if p.GetPatchType() != types.JSONPatchType {
				return true, live, nil
			}
			require.Contains(t, string(p.GetPatch()), `"manager":"terraform-provider-akp","operation":"Apply"`)
			require.NotContains(t, string(p.GetPatch()), live.GetManagedFields()[0].Manager)
			if conflicts > 0 {
				conflicts--
				return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, "agent", nil)
			}
			return true, live, nil
		})
		err := client.Apply(ctx, cm)
		return sent, err
	}

	t.Run("fresh object is applied once", func(t *testing.T) {
		sent, err := run(t, cm, 0)
		require.NoError(t, err)
		require.Equal(t, []types.PatchType{types.ApplyPatchType}, sent)
	})

	for _, manager := range []string{"HashiCorp", "kubectl-client-side-apply"} {
		t.Run(manager, func(t *testing.T) {
			csaOwned := cm.DeepCopy()
			csaOwned.SetResourceVersion("7")
			csaOwned.SetAnnotations(map[string]string{corev1.LastAppliedConfigAnnotation: "{}"})
			csaOwned.SetManagedFields([]metav1.ManagedFieldsEntry{{
				Manager:    manager,
				Operation:  metav1.ManagedFieldsOperationUpdate,
				APIVersion: "v1",
				FieldsType: "FieldsV1",
				FieldsV1:   &metav1.FieldsV1{Raw: []byte(`{"f:data":{"f:stale":{}},"f:metadata":{"f:annotations":{"f:kubectl.kubernetes.io/last-applied-configuration":{}}}}`)},
			}})
			for name, tc := range map[string]struct {
				conflicts int
				want      []types.PatchType
				wantError bool
			}{
				"migrated then re-applied": {
					want: []types.PatchType{types.ApplyPatchType, types.JSONPatchType, types.ApplyPatchType},
				},
				"retries on a resourceVersion conflict": {
					conflicts: 1,
					want:      []types.PatchType{types.ApplyPatchType, types.JSONPatchType, types.JSONPatchType, types.ApplyPatchType},
				},
				"last migration attempt succeeds": {
					conflicts: 4,
					want:      []types.PatchType{types.ApplyPatchType, types.JSONPatchType, types.JSONPatchType, types.JSONPatchType, types.JSONPatchType, types.JSONPatchType, types.ApplyPatchType},
				},
				"exhausted migration conflicts surface": {
					conflicts: 5,
					want:      []types.PatchType{types.ApplyPatchType, types.JSONPatchType, types.JSONPatchType, types.JSONPatchType, types.JSONPatchType, types.JSONPatchType},
					wantError: true,
				},
			} {
				t.Run(name, func(t *testing.T) {
					sent, err := run(t, csaOwned, tc.conflicts)
					if tc.wantError {
						require.True(t, apierrors.IsConflict(err), "got %v", err)
					} else {
						require.NoError(t, err)
					}
					require.Equal(t, tc.want, sent)
				})
			}
		})
	}
}
