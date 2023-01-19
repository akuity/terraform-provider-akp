package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/openapi"
)

// NewKubectl returns a kubectl instance from a rest config
func NewKubectl(config *rest.Config) (*Kubectl, error) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.WithWrapConfigFn(func(_ *rest.Config) *rest.Config {
		return config
	})
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	fact := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	return &Kubectl{
		config: config,
		fact:   fact,
	}, nil
}

type Kubectl struct {
	config        *rest.Config
	fact          cmdutil.Factory
	openAPISchema openapi.Resources
}

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

func (k *Kubectl) OpenAPISchema() (openapi.Resources, error) {
	if k.openAPISchema != nil {
		return k.openAPISchema, nil
	}
	disco, err := discovery.NewDiscoveryClientForConfig(k.config)
	if err != nil {
		return nil, err
	}
	openAPISchema, err := openapi.NewOpenAPIParser(openapi.NewOpenAPIGetter(disco)).Parse()
	if err != nil {
		return nil, err
	}
	k.openAPISchema = openAPISchema
	return k.openAPISchema, nil
}

func (k *Kubectl) newApplyOptions(ioStreams genericclioptions.IOStreams, obj *unstructured.Unstructured, path string, applyOpts ApplyOpts) (*apply.ApplyOptions, error) {
	flags := apply.NewApplyFlags(k.fact, ioStreams)
	o := &apply.ApplyOptions{
		IOStreams:         ioStreams,
		VisitedUids:       sets.NewString(),
		VisitedNamespaces: sets.NewString(),
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
	o.OpenAPISchema, err = k.OpenAPISchema()
	if err != nil {
		return nil, err
	}
	o.DryRunVerifier = resource.NewQueryParamVerifier(dynamicClient, k.fact.OpenAPIGetter(), resource.QueryParamFieldValidation)
	validateDirective := metav1.FieldValidationIgnore
	if applyOpts.Validate {
		validateDirective = metav1.FieldValidationStrict
	}
	o.Validator, err = k.fact.Validator(validateDirective, o.DryRunVerifier)
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

func writeFile(bytes []byte) (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", fmt.Errorf("Failed to generate temp file for manifest: %v", err)
	}
	if _, err = f.Write(bytes); err != nil {
		return "", fmt.Errorf("Failed to write manifest: %v", err)
	}
	if err = f.Close(); err != nil {
		return "", fmt.Errorf("Failed to close manifest: %v", err)
	}
	return f.Name(), nil
}

func deleteFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	_ = os.Remove(path)
}
