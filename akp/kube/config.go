package kube

import (
	"fmt"

	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/openapi"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

type Kubectl struct {
	config        *rest.Config
	fact          cmdutil.Factory
	openAPISchema openapi.Resources
}

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

// Adapted github.com/gavinbunney/terraform-provider-kubectl/kubernetes/provider.go functions

func InitializeConfiguration(k *types.Kubeconfig) (*rest.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}
	configPaths := []string{}
	if v := k.ConfigPath.ValueString(); v != "" {
		configPaths = []string{v}
	} else if v := k.ConfigPaths.Elements(); len(v) > 0 {
		for _, p := range v {
			configPaths = append(configPaths, p.String())
		}
	}
	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxSuffix := "; default context"

		kubectx := k.ConfigContext.ValueString()
		authInfo := k.ConfigContextAuthInfo.ValueString()
		cluster := k.ConfigContextCluster.ValueString()
		if kubectx != "" || authInfo != "" || cluster != "" {
			ctxSuffix = "; overriden context"
			if kubectx != "" {
				overrides.CurrentContext = kubectx
				ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfo != "" {
				overrides.Context.AuthInfo = authInfo
				ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if cluster != "" {
				overrides.Context.Cluster = cluster
				ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
		}
	}
	// Overriding with static configuration
	if !k.Insecure.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = k.Insecure.ValueBool()
	}
	if v := k.ClusterCaCertificate.ValueString(); v != "" {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(v)
	}
	if v := k.ClientCertificate.ValueString(); v != "" {
		overrides.AuthInfo.ClientCertificateData = []byte(v)
	}
	if v := k.Host.ValueString(); v != "" {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(v, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if v := k.Username.ValueString(); v != "" {
		overrides.AuthInfo.Username = v
	}
	if v := k.Password.ValueString(); v != "" {
		overrides.AuthInfo.Password = v
	}
	if v := k.ClientKey.ValueString(); v != "" {
		overrides.AuthInfo.ClientKeyData = []byte(v)
	}
	if v := k.Token.ValueString(); v != "" {
		overrides.AuthInfo.Token = v
	}
	if v := k.ProxyUrl.ValueString(); v != "" {
		overrides.ClusterDefaults.ProxyURL = v
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("Invalid provider configuration: %s", err)
	}
	cfg.QPS = 100.0
	cfg.Burst = 100

	// Overriding with static configuration
	terraformVersion := "unknown"
	cfg.UserAgent = fmt.Sprintf("HashiCorp/1.0 Terraform/%s", terraformVersion)
	return cfg, nil
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
