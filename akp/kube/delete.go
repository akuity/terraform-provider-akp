package kube

import (
	"bytes"
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/util"
)

type DeleteOpts struct {
	Force           bool
	WaitForDeletion bool
	IgnoreNotFound  bool
	GracePeriod     int
}

func (k *Kubectl) DeleteResource(ctx context.Context, obj *unstructured.Unstructured, deleteOpts DeleteOpts) (string, error) {
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	path, err := writeFile(objBytes)
	if err != nil {
		return "", err
	}
	defer deleteFile(path)
	dcmd := delete.NewCmdDelete(k.fact, ioStreams)
	kubeDeleteOpts, err := k.newDeleteOptions(ioStreams, obj, path, deleteOpts)
	if err != nil {
		return "", err
	}
	err = kubeDeleteOpts.Complete(k.fact, []string{}, dcmd)
	if err != nil {
		return "", err
	}
	err = kubeDeleteOpts.RunDelete(k.fact)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (k *Kubectl) newDeleteOptions(ioStreams genericclioptions.IOStreams, obj *unstructured.Unstructured, path string, deleteOpts DeleteOpts) (*delete.DeleteOptions, error) {
	o := &delete.DeleteOptions{
		FilenameOptions: resource.FilenameOptions{
			Filenames: []string{path},
		},
		IgnoreNotFound:    deleteOpts.IgnoreNotFound,
		WaitForDeletion:   deleteOpts.WaitForDeletion,
		GracePeriod:       deleteOpts.GracePeriod,
		Output:            "name",
		IOStreams:         ioStreams,
		CascadingStrategy: metav1.DeletePropagationBackground,
	}
	dynamicClient, err := dynamic.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}
	o.DynamicClient = dynamicClient
	o.DryRunStrategy = util.DryRunNone
	return o, nil
}
