package kube

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/util/csaupgrade"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

const fieldManager = "terraform-provider-akp"

var deleteTimeout = 10 * time.Minute

type Client struct {
	dyn    dynamic.Interface
	mapper *restmapper.DeferredDiscoveryRESTMapper
}

func NewClient(cfg *rest.Config) (*Client, error) {
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{dyn: dyn, mapper: restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))}, nil
}

var (
	applyOpts       = metav1.PatchOptions{FieldManager: fieldManager, Force: new(true)}
	lastAppliedPath = fieldpath.NewSet(fieldpath.MakePathOrDie("metadata", "annotations", corev1.LastAppliedConfigAnnotation))
)

// Apply performs a server-side apply of obj, creating it when missing and taking
// ownership of any conflicting fields. Objects still owned by an older provider's
// client-side apply get that ownership migrated and are applied once more so
// fields dropped from the manifest are pruned, mirroring kubectl apply --server-side.
func (c *Client) Apply(ctx context.Context, obj *unstructured.Unstructured) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	ri, err := c.resource(obj)
	if err != nil {
		return err
	}
	live, err := ri.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, applyOpts)
	if err != nil {
		return err
	}
	migrated, err := migrateClientSideApply(ctx, ri, live)
	if err != nil || !migrated {
		return err
	}
	_, err = ri.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, applyOpts)
	return err
}

// migrateClientSideApply hands the fields owned by client-side-apply managers to
// our field manager. The patch is guarded by resourceVersion, so it retries when a
// concurrent write (typically a status update) lands in between.
func migrateClientSideApply(ctx context.Context, ri dynamic.ResourceInterface, live *unstructured.Unstructured) (bool, error) {
	managers := sets.New[string]()
	for _, e := range csaupgrade.FindFieldsOwners(live.GetManagedFields(), metav1.ManagedFieldsOperationUpdate, lastAppliedPath) {
		managers.Insert(e.Manager)
	}
	var err error
	for range 5 {
		var patch []byte
		if patch, err = csaupgrade.UpgradeManagedFieldsPatch(live, managers, fieldManager); err != nil || patch == nil {
			return false, err
		}
		if _, err = ri.Patch(ctx, live.GetName(), types.JSONPatchType, patch, metav1.PatchOptions{}); err == nil {
			return true, nil
		}
		if !errors.IsConflict(err) {
			return false, err
		}
		var getErr error
		if live, getErr = ri.Get(ctx, live.GetName(), metav1.GetOptions{}); getErr != nil {
			return false, getErr
		}
	}
	return false, err
}

// Delete removes obj, tolerating a missing object, and waits until it is gone.
func (c *Client) Delete(ctx context.Context, obj *unstructured.Unstructured) error {
	// Terraform does not necessarily supply a deadline. A stuck finalizer must
	// not keep destroy waiting forever; an earlier caller deadline still wins.
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()
	ri, err := c.resource(obj)
	if err != nil {
		return err
	}
	err = ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{PropagationPolicy: new(metav1.DeletePropagationBackground)})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := ri.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func (c *Client) resource(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gvk := obj.GroupVersionKind()
	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if meta.IsNoMatchError(err) {
		// A CRD applied earlier in the same batch is not in the cached discovery yet.
		c.mapper.Reset()
		mapping, err = c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}
	if err != nil {
		return nil, err
	}
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		return c.dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace()), nil
	}
	return c.dyn.Resource(mapping.Resource), nil
}
