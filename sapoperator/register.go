package sapoperator

import (
	"github.com/SAP/sap-btp-service-operator/api/v1alpha1"
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const SVCATGroupName = "servicecatalog.k8s.io"
const SVCATGroupVersion = "v1beta1"
const OperatorGroupName = "services.cloud.sap.com"
const OperatorGroupVersion = "v1alpha1"

var SvcatSchemeGroupVersion = schema.GroupVersion{Group: SVCATGroupName, Version: SVCATGroupVersion}
var OperatorSchemeGroupVersion = schema.GroupVersion{Group: OperatorGroupName, Version: OperatorGroupVersion}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SvcatSchemeGroupVersion,
		&v1beta1.ServiceInstance{},
		&v1beta1.ServiceInstanceList{},
		&v1beta1.ServiceBinding{},
		&v1beta1.ServiceBindingList{},
	)
	scheme.AddKnownTypes(OperatorSchemeGroupVersion,
		&v1alpha1.ServiceInstance{},
		&v1alpha1.ServiceInstanceList{},
		&v1alpha1.ServiceBinding{},
		&v1alpha1.ServiceBindingList{},
	)

	metav1.AddToGroupVersion(scheme, SvcatSchemeGroupVersion)
	metav1.AddToGroupVersion(scheme, OperatorSchemeGroupVersion)
	return nil
}
