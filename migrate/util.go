package migrate

import (
	"context"
	"github.com/SAP/sap-btp-service-operator/client/sm"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func GetSMClient(ctx context.Context, secret *v1.Secret) sm.Client {
	secretData := secret.Data
	return sm.NewClient(ctx, &sm.ClientConfig{
		ClientID:     string(secretData["clientid"]),
		ClientSecret: string(secretData["clientsecret"]),
		URL:          string(secretData["url"]),
		TokenURL:     string(secretData["tokenurl"]),
		SSLDisabled:  false,
	}, nil)
}

func GetK8sClient(config *rest.Config, groupName, groupVersion string) *rest.RESTClient {
	opcrdConfig := *config
	opcrdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: groupName, Version: groupVersion}
	opcrdConfig.APIPath = "/apis"
	opcrdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	opcrdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	operatorClient, err := rest.UnversionedRESTClientFor(&opcrdConfig)
	cobra.CheckErr(err)
	return operatorClient
}
