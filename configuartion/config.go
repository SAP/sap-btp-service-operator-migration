package configuartion

import (
	"context"
	"github.com/spf13/viper"
)

// Configuration contains the information that will be saved/loaded in the CLI config file
type Configuration struct {
	Context          context.Context
	ManagedNamespace string
	KubeConfig       string
}

func NewConfiguration(ctx context.Context, env *viper.Viper) *Configuration {

	return &Configuration{
		Context:          ctx,
		ManagedNamespace: env.Get("managedNamespace").(string),
		KubeConfig:       env.Get("kubeconfig").(string),
	}
}
