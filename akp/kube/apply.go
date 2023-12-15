package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type ApplyOpts struct {
	DryRunStrategy cmdutil.DryRunStrategy
	Force          bool
	Validate       bool
}

// ApplyResource performs an apply of a unstructured resource
func (k *Kubectl) ApplyResource(ctx context.Context, obj *unstructured.Unstructured, applyOpts ApplyOpts) (string, error) {
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
	kubeApplyOpts, err := k.newApplyOptions(ioStreams, obj, path, applyOpts)
	if err != nil {
		return "", err
	}
	applyErr := kubeApplyOpts.Run()
	var out []string
	if buf := strings.TrimSpace(ioStreams.Out.(*bytes.Buffer).String()); len(buf) > 0 {
		out = append(out, buf)
	}
	if buf := strings.TrimSpace(ioStreams.ErrOut.(*bytes.Buffer).String()); len(buf) > 0 {
		out = append(out, buf)
	}
	return strings.Join(out, ". "), applyErr
}

func (k *Kubectl) newApplyOptions(ioStreams genericclioptions.IOStreams, obj *unstructured.Unstructured, path string, applyOpts ApplyOpts) (*apply.ApplyOptions, error) {
	flags := apply.NewApplyFlags(ioStreams)
	o := &apply.ApplyOptions{
		IOStreams:         ioStreams,
		VisitedUids:       sets.New[types.UID](),
		VisitedNamespaces: sets.New[string](),
		Recorder:          genericclioptions.NoopRecorder{},
		PrintFlags:        flags.PrintFlags,
		Overwrite:         true,
		OpenAPIPatch:      true,
	}
	dynamicClient, err := dynamic.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}
	o.DynamicClient = dynamicClient
	o.DeleteOptions, err = delete.NewDeleteFlags("").ToOptions(dynamicClient, ioStreams)
	if err != nil {
		return nil, err
	}
	o.OpenAPIGetter = k

	validateDirective := metav1.FieldValidationIgnore
	if applyOpts.Validate {
		validateDirective = metav1.FieldValidationStrict
	}
	o.Validator, err = k.fact.Validator(validateDirective)
	if err != nil {
		return nil, err
	}
	o.Builder = k.fact.NewBuilder()
	o.Mapper, err = k.fact.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	o.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		o.PrintFlags.NamePrintFlags.Operation = operation
		switch o.DryRunStrategy {
		case cmdutil.DryRunClient:
			err = o.PrintFlags.Complete("%s (dry run)")
			if err != nil {
				return nil, err
			}
		case cmdutil.DryRunServer:
			err = o.PrintFlags.Complete("%s (server dry run)")
			if err != nil {
				return nil, err
			}
		}
		return o.PrintFlags.ToPrinter()
	}
	o.DeleteOptions.FilenameOptions.Filenames = []string{path}
	o.Namespace = obj.GetNamespace()
	o.DeleteOptions.ForceDeletion = applyOpts.Force
	o.DryRunStrategy = applyOpts.DryRunStrategy
	return o, nil
}
