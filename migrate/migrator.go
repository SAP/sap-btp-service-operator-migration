package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/SAP/sap-btp-service-operator-migration/sapoperator"
	"github.com/SAP/sap-btp-service-operator/api/v1alpha1"
	"github.com/SAP/sap-btp-service-operator/client/sm"
	"github.com/SAP/sap-btp-service-operator/client/sm/types"
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	migratedLabel = "migrated"
)

type Migrator struct {
	SMClient              sm.Client
	SvcatRestClient       *rest.RESTClient
	SapOperatorRestClient *rest.RESTClient
	ClientSet             *kubernetes.Clientset
	ClusterID             string
	Services              map[string]types.ServiceOffering
	Plans                 map[string]types.ServicePlan
}

type serviceInstancePair struct {
	svcatInstance *v1beta1.ServiceInstance
	smInstance    *types.ServiceInstance
}

type serviceBindingPair struct {
	svcatBinding *v1beta1.ServiceBinding
	smBinding    *types.ServiceBinding
}

type object interface {
	metav1.Object
	runtime.Object
}

type ExecutionMode int

const (
	Run ExecutionMode = iota
	RunWithoutValidation
	DryRun
)

const ServiceInstances = "serviceinstances"
const ServiceBindings = "servicebindings"

func NewMigrator(ctx context.Context, kubeconfig string, managedNamespace string) *Migrator {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	cobra.CheckErr(err)

	err = sapoperator.AddToScheme(scheme.Scheme)
	cobra.CheckErr(err)

	clientset, err := kubernetes.NewForConfig(config)
	cobra.CheckErr(err)

	secret, err := clientset.CoreV1().Secrets(managedNamespace).Get(ctx, "sap-btp-service-operator", metav1.GetOptions{})
	cobra.CheckErr(err)

	configMap, err := clientset.CoreV1().ConfigMaps(managedNamespace).Get(ctx, "sap-btp-operator-config", metav1.GetOptions{})
	cobra.CheckErr(err)

	return getMigrator(
		GetSMClient(ctx, secret),
		GetK8sClient(config, sapoperator.SVCATGroupName, sapoperator.SVCATGroupVersion),
		GetK8sClient(config, sapoperator.OperatorGroupName, sapoperator.OperatorGroupVersion),
		configMap.Data["CLUSTER_ID"],
		clientset,
	)
}

func getMigrator(smClient sm.Client, svcatRestClient, sapOperatorRestClient *rest.RESTClient, clusterID string, clientset *kubernetes.Clientset) *Migrator {
	fmt.Println(fmt.Sprintf("Migrator initialized with cluster ID '%s'", clusterID))
	return &Migrator{
		SMClient:              smClient,
		SvcatRestClient:       svcatRestClient,
		SapOperatorRestClient: sapOperatorRestClient,
		ClientSet:             clientset,
		ClusterID:             clusterID,
		Services:              getServices(smClient),
		Plans:                 getPlans(smClient),
	}
}

func (m *Migrator) Migrate(ctx context.Context, executionMode ExecutionMode) {
	parameters := &sm.Parameters{
		FieldQuery: []string{
			fmt.Sprintf("context/clusterid eq '%s'", m.ClusterID),
		},
	}

	smInstances, err := m.SMClient.ListInstances(parameters)
	cobra.CheckErr(err)
	fmt.Println(fmt.Sprintf("*** Fetched %v instances from SM", len(smInstances.ServiceInstances)))

	smBindings, err := m.SMClient.ListBindings(parameters)
	cobra.CheckErr(err)
	fmt.Println(fmt.Sprintf("*** Fetched %v bindings from SM", len(smBindings.ServiceBindings)))

	svcatInstances := v1beta1.ServiceInstanceList{}
	err = m.SvcatRestClient.Get().Namespace("").Resource(ServiceInstances).Do(ctx).Into(&svcatInstances)
	cobra.CheckErr(err)
	fmt.Println(fmt.Sprintf("*** Fetched %v svcat instances from cluster", len(svcatInstances.Items)))

	svcatBindings := v1beta1.ServiceBindingList{}
	err = m.SvcatRestClient.Get().Namespace("").Resource(ServiceBindings).Do(ctx).Into(&svcatBindings)
	cobra.CheckErr(err)
	fmt.Println(fmt.Sprintf("*** Fetched %v svcat bindings from cluster", len(svcatBindings.Items)))

	fmt.Println("*** Preparing resources")
	instancesToMigrate := m.getInstancesToMigrate(smInstances, svcatInstances)
	bindingsToMigrate := m.getBindingsToMigrate(smBindings, svcatBindings)
	if len(instancesToMigrate) == 0 && len(bindingsToMigrate) == 0 {
		fmt.Println("no svcat instances or bindings found for migration")
		return
	}
	fmt.Println(fmt.Sprintf("*** found %d instances and %d bindings to migrate", len(instancesToMigrate), len(bindingsToMigrate)))

	if executionMode != RunWithoutValidation {
		fmt.Println("*** Validating")
		failuresCount, validationErrorsMsg := m.validate(ctx, instancesToMigrate, bindingsToMigrate)
		if failuresCount > 0 {
			fmt.Println(fmt.Sprintf("Validation failed got %d validation errors:", failuresCount))
			cobra.CheckErr(validationErrorsMsg)
		} else {
			fmt.Println("*** Validation completed successfully")
		}
		if executionMode == DryRun {
			return
		}
	} else {
		fmt.Println("*** Validation is skipped...")
	}

	var failuresBuffer bytes.Buffer
	for _, pair := range instancesToMigrate {
		err := m.migrateInstance(ctx, pair)
		if err != nil {
			fmt.Println(err.Error())
			failuresBuffer.WriteString(err.Error() + "\n")
		}
	}

	for _, pair := range bindingsToMigrate {
		err := m.migrateBinding(ctx, pair)
		if err != nil {
			fmt.Println(err.Error())
			failuresBuffer.WriteString(err.Error() + "\n")
		}
	}

	if failuresBuffer.Len() == 0 {
		fmt.Println("*** Migration completed successfully")
	} else {
		fmt.Println("*** Migration failures summary:")
		cobra.CheckErr(failuresBuffer.String())
	}
}

func (m *Migrator) getInstancesToMigrate(smInstances *types.ServiceInstances, svcatInstances v1beta1.ServiceInstanceList) []serviceInstancePair {
	validInstances := make([]serviceInstancePair, 0)
	for _, svcat := range svcatInstances.Items {
		var smInstance *types.ServiceInstance
		for _, instance := range smInstances.ServiceInstances {
			if instance.ID == svcat.Spec.ExternalID {
				smInstance = &instance
				break
			}
		}
		if smInstance == nil {
			fmt.Println(fmt.Sprintf("svcat instance name '%s' id '%s' (%s) not found in SM, skipping it...", svcat.Name, svcat.Spec.ExternalID, svcat.Name))
			continue
		}
		svcInstance := svcat
		validInstances = append(validInstances, serviceInstancePair{
			svcatInstance: &svcInstance,
			smInstance:    smInstance,
		})
	}

	return validInstances
}

func (m *Migrator) getBindingsToMigrate(smBindings *types.ServiceBindings, svcatBindings v1beta1.ServiceBindingList) []serviceBindingPair {
	validBindings := make([]serviceBindingPair, 0)
	for _, svcat := range svcatBindings.Items {
		var smBinding *types.ServiceBinding
		for _, binding := range smBindings.ServiceBindings {
			if binding.ID == svcat.Spec.ExternalID {
				smBinding = &binding
				break
			}
		}
		if smBinding == nil {
			fmt.Println(fmt.Sprintf("svcat binding name '%s' id '%s' (%s) not found in SM, skipping it...", svcat.Name, svcat.Spec.ExternalID, svcat.Name))
			continue
		}
		svcBinding := svcat
		validBindings = append(validBindings, serviceBindingPair{
			svcatBinding: &svcBinding,
			smBinding:    smBinding,
		})
	}

	return validBindings
}

func (m *Migrator) migrateInstanceDryRun(ctx context.Context, pair serviceInstancePair) error {
	instance := m.getInstanceStruct(pair)
	err := m.SapOperatorRestClient.Post().
		Namespace(pair.svcatInstance.Namespace).
		Resource(ServiceInstances).
		Param("dryRun", "All").
		Body(instance).
		Do(ctx).
		Error()
	return m.ignoreAlreadyMigrated(ctx, instance, &v1alpha1.ServiceInstance{}, err)
}

func (m *Migrator) migrateBindingDryRun(ctx context.Context, pair serviceBindingPair) error {
	binding := m.getBindingStruct(pair)
	err := m.SapOperatorRestClient.Post().
		Namespace(pair.svcatBinding.Namespace).
		Resource(ServiceBindings).
		Param("dryRun", "All").
		Body(binding).
		Do(ctx).
		Error()
	return m.ignoreAlreadyMigrated(ctx, binding, &v1alpha1.ServiceBinding{}, err)
}

func (m *Migrator) migrateInstance(ctx context.Context, pair serviceInstancePair) error {

	fmt.Println(fmt.Sprintf("migrating service instance '%s' in namespace '%s' (smID: '%s')", pair.svcatInstance.Name, pair.svcatInstance.Namespace, pair.svcatInstance.Spec.ExternalID))

	//set k8s label
	requestBody := fmt.Sprintf(`{"k8sname": "%s"}`, pair.svcatInstance.Name)
	buffer := bytes.NewBuffer([]byte(requestBody))
	response, err := m.SMClient.Call(http.MethodPut, fmt.Sprintf("/v1/migrate/service_instances/%s", pair.smInstance.ID), buffer, &sm.Parameters{})
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			fmt.Println(response.StatusCode)
		}
		return fmt.Errorf("failed to add k8s label to service instance name: %s, ID: %s", pair.smInstance.Name, pair.smInstance.ID)
	}

	instance := m.getInstanceStruct(pair)
	res := &v1alpha1.ServiceInstance{}
	err = m.SapOperatorRestClient.Post().
		Namespace(pair.svcatInstance.Namespace).
		Resource(ServiceInstances).
		Body(instance).
		Do(ctx).
		Into(res)

	if err = m.ignoreAlreadyMigrated(ctx, instance, res, err); err != nil {
		return fmt.Errorf("failed to create service instance: %v", err.Error())
	}

	if !pair.svcatInstance.DeletionTimestamp.IsZero() {
		fmt.Println(fmt.Sprintf("svcat instance '%s' is marked for deletion, deleting it from operator", pair.svcatInstance.Name))
		err = m.SapOperatorRestClient.Delete().Name(res.Name).Namespace(res.Namespace).Do(ctx).Error()
		if err != nil {
			fmt.Println(fmt.Sprintf("failed to delete instance from operator: %v", err.Error()))
		}
	}

	pair.svcatInstance.Finalizers = []string{}
	err = m.SvcatRestClient.Put().Name(pair.svcatInstance.Name).Namespace(pair.svcatInstance.Namespace).Resource(ServiceInstances).Body(pair.svcatInstance).Do(ctx).Error()
	if err != nil {
		return fmt.Errorf("failed to delete finalizer from instance '%s'. Error: %v", pair.svcatInstance.Name, err.Error())
	}

	err = m.deleteSvcatResource(ctx, res.Name, res.Namespace, ServiceInstances)
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to delete svcat resource. Error: %v", err.Error()))
	}
	fmt.Println("instance migrated successfully")
	return nil
}

func (m *Migrator) migrateBinding(ctx context.Context, pair serviceBindingPair) error {

	fmt.Println(fmt.Sprintf("migrating service binding '%s' in namespace '%s' (smID: '%s')", pair.svcatBinding.Name, pair.svcatBinding.Namespace, pair.svcatBinding.Spec.ExternalID))
	secretExists := true
	secret, err := m.ClientSet.CoreV1().Secrets(pair.svcatBinding.Namespace).Get(ctx, pair.svcatBinding.Spec.SecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println(fmt.Sprintf("Info: secret named '%s' not found for binding", pair.svcatBinding.Spec.SecretName))
			secretExists = false
		} else {
			return fmt.Errorf("failed to get binding's secret, skipping binding migration. Error: %v", err.Error())
		}
	}
	//add k8sname label and save credentials
	requestBody, err := m.getMigrateBindingRequestBody(pair.svcatBinding.Name, secret)
	if err != nil {
		return fmt.Errorf("failed to build request body for migrating instance. Error: %v", err.Error())
	}
	buffer := bytes.NewBuffer([]byte(requestBody))
	response, err := m.SMClient.Call(http.MethodPut, fmt.Sprintf("/v1/migrate/service_bindings/%s", pair.smBinding.ID), buffer, &sm.Parameters{})
	if err != nil || response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add k8s label to service binding name: %s, ID: %s", pair.smBinding.Name, pair.smBinding.ID)
	}

	if secretExists {
		//add 'binding' label to secret
		if secret.Labels == nil {
			secret.Labels = make(map[string]string, 1)
		}
		secret.Labels["binding"] = pair.svcatBinding.Name
		secret, err = m.ClientSet.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to add label to binding. Error: %v", err.Error())
		}
	}

	binding := m.getBindingStruct(pair)
	res := &v1alpha1.ServiceBinding{}
	err = m.SapOperatorRestClient.Post().
		Namespace(binding.Namespace).
		Resource(ServiceBindings).
		Body(binding).
		Do(ctx).
		Into(res)
	if err = m.ignoreAlreadyMigrated(ctx, binding, res, err); err != nil {
		return fmt.Errorf("failed to create service binding: %v", err.Error())
	}

	if secretExists {
		//set the new binding as owner reference for the secret
		t := true
		owner := metav1.OwnerReference{
			APIVersion:         res.APIVersion,
			Kind:               res.Kind,
			Name:               res.Name,
			UID:                res.UID,
			Controller:         &t,
			BlockOwnerDeletion: &t,
		}
		secret.OwnerReferences = []metav1.OwnerReference{owner}
		secret, err = m.ClientSet.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to set new binding as owner of secret. Error: %v", err.Error())
		}
	}

	if !pair.svcatBinding.DeletionTimestamp.IsZero() {
		fmt.Println(fmt.Sprintf("svcat binding '%s' is marked for deletion, deleting it from operator", pair.svcatBinding.Name))
		err = m.SapOperatorRestClient.Delete().Name(res.Name).Namespace(res.Namespace).Do(ctx).Error()
		if err != nil {
			fmt.Println(fmt.Sprintf("failed to delete binding from operator. Error: %v", err.Error()))
		}
	}

	//remove finalizer from binding to avoid deletion of the secret
	pair.svcatBinding.Finalizers = []string{}
	err = m.SvcatRestClient.Put().Name(pair.svcatBinding.Name).Namespace(pair.svcatBinding.Namespace).Resource(ServiceBindings).Body(pair.svcatBinding).Do(ctx).Error()
	if err != nil {
		return fmt.Errorf("failed to delete finalizer from binding '%s'. Error: %v", pair.svcatBinding.Name, err.Error())
	}

	err = m.deleteSvcatResource(ctx, res.Name, res.Namespace, ServiceBindings)
	if err != nil {
		return fmt.Errorf("failed to delete svcat binding. Error: %v", err.Error())
	}
	fmt.Println("binding migrated successfully")
	return nil
}

func (m *Migrator) deleteSvcatResource(ctx context.Context, resourceName string, resourceNamespace string, resourceType string) error {

	err := m.SapOperatorRestClient.Get().Name(resourceName).Namespace(resourceNamespace).Resource(resourceType).Do(ctx).Error()
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to get the migrated service instance '%s' status, corresponding svcat resource will not be deleted. Error: %v",
			resourceName, err.Error()))
		return err
	}

	//fmt.Println(fmt.Sprintf("deleting svcat resource type '%s' named '%s' in namespace '%s'", resourceType, resourceName, resourceNamespace))
	err = m.SvcatRestClient.Delete().Name(resourceName).Namespace(resourceNamespace).Resource(resourceType).Do(ctx).Error()
	return err
}

func (m *Migrator) getMigrateBindingRequestBody(k8sName string, secret *corev1.Secret) (string, error) {
	var err error
	secretData := []byte("")
	secretDataEncoded := make(map[string]string)
	if secret != nil {
		for k, v := range secret.Data {
			secretDataEncoded[k] = string(v)
		}

		secretData, err = json.Marshal(secretDataEncoded)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf(`
		{
			"k8sname": "%s",
			"credentials": %s
		}`, k8sName, secretData), nil
}

func (m *Migrator) validate(ctx context.Context, instancesToMigrate []serviceInstancePair, bindingsToMigrate []serviceBindingPair) (int, *bytes.Buffer) {
	buffer := new(bytes.Buffer)
	count := 0
	for _, pair := range instancesToMigrate {
		err := m.migrateInstanceDryRun(ctx, pair)
		if err != nil {
			count++
			buffer.WriteString(fmt.Sprintf("instance '%s' in namespace '%s' failed: '%v' \n", pair.svcatInstance.Name, pair.svcatInstance.Namespace, err.Error()))
		} else {
			fmt.Println(fmt.Sprintf("svcat instance '%s' in namespace '%s' was validated successfully", pair.svcatInstance.Name, pair.svcatInstance.Namespace))
		}
	}

	for _, pair := range bindingsToMigrate {
		err := m.migrateBindingDryRun(ctx, pair)
		if err != nil {
			count++
			buffer.WriteString(fmt.Sprintf("binding '%s' in namespace '%s' failed: '%v' \n", pair.svcatBinding.Name, pair.svcatBinding.Namespace, err.Error()))
		} else {
			fmt.Println(fmt.Sprintf("svcat binding '%s' in namespace '%s' was validated successfully", pair.svcatBinding.Name, pair.svcatBinding.Namespace))
		}
	}
	return count, buffer
}

func getPlans(smclient sm.Client) map[string]types.ServicePlan {
	plans, err := smclient.ListPlans(nil)
	cobra.CheckErr(err)
	res := make(map[string]types.ServicePlan)
	for _, plan := range plans.ServicePlans {
		res[plan.ID] = plan
	}
	return res
}

func getServices(smclient sm.Client) map[string]types.ServiceOffering {
	services, err := smclient.ListOfferings(nil)
	cobra.CheckErr(err)
	res := make(map[string]types.ServiceOffering)
	for _, svc := range services.ServiceOfferings {
		res[svc.ID] = svc
	}
	return res
}

func (m *Migrator) getInstanceStruct(pair serviceInstancePair) *v1alpha1.ServiceInstance {
	plan := m.Plans[pair.smInstance.ServicePlanID]
	service := m.Services[plan.ServiceOfferingID]

	parametersFrom := make([]v1alpha1.ParametersFromSource, 0)
	for _, param := range pair.svcatInstance.Spec.ParametersFrom {
		parametersFrom = append(parametersFrom, v1alpha1.ParametersFromSource{
			SecretKeyRef: &v1alpha1.SecretKeyReference{
				Name: param.SecretKeyRef.Name,
				Key:  param.SecretKeyRef.Key,
			},
		})
	}

	userInfo, err := json.Marshal(pair.svcatInstance.Spec.UserInfo)
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to parse user info for instance %s: %v", pair.svcatInstance.Name, err.Error()))
	}

	return &v1alpha1.ServiceInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s", sapoperator.OperatorGroupName, sapoperator.OperatorGroupVersion),
			Kind:       "ServiceInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pair.svcatInstance.Name,
			Namespace: pair.svcatInstance.Namespace,
			Labels: map[string]string{
				migratedLabel: "true",
			},
			Annotations: map[string]string{
				"original_creation_timestamp": pair.svcatInstance.CreationTimestamp.String(),
				"original_user_info":          string(userInfo)},
		},
		Spec: v1alpha1.ServiceInstanceSpec{
			ServicePlanName:     plan.Name,
			ServiceOfferingName: service.Name,
			ExternalName:        pair.smInstance.Name,
			ParametersFrom:      parametersFrom,
			Parameters:          pair.svcatInstance.Spec.Parameters,
		},
	}
}

func (m *Migrator) getBindingStruct(pair serviceBindingPair) *v1alpha1.ServiceBinding {
	parametersFrom := make([]v1alpha1.ParametersFromSource, 0)
	for _, param := range pair.svcatBinding.Spec.ParametersFrom {
		parametersFrom = append(parametersFrom, v1alpha1.ParametersFromSource{
			SecretKeyRef: &v1alpha1.SecretKeyReference{
				Name: param.SecretKeyRef.Name,
				Key:  param.SecretKeyRef.Key,
			},
		})
	}

	userInfo, err := json.Marshal(pair.svcatBinding.Spec.UserInfo)
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to parse user info for binding %s. Error: %v", pair.svcatBinding.Name, err.Error()))
	}

	return &v1alpha1.ServiceBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s", sapoperator.OperatorGroupName, sapoperator.OperatorGroupVersion),
			Kind:       "ServiceBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pair.svcatBinding.Name,
			Namespace: pair.svcatBinding.Namespace,
			Labels: map[string]string{
				migratedLabel: "true",
			},
			Annotations: map[string]string{
				"original_creation_timestamp": pair.svcatBinding.CreationTimestamp.String(),
				"original_user_info":          string(userInfo)},
		},
		Spec: v1alpha1.ServiceBindingSpec{
			ServiceInstanceName: pair.svcatBinding.Spec.InstanceRef.Name,
			ExternalName:        pair.smBinding.Name,
			ParametersFrom:      parametersFrom,
			Parameters:          pair.svcatBinding.Spec.Parameters,
		},
	}
}

func (m *Migrator) ignoreAlreadyMigrated(ctx context.Context, obj, res object, err error) error {
	if err == nil {
		return nil
	}
	if !errors.IsAlreadyExists(err) {
		return err
	}
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	resource := fmt.Sprintf("%vs", strings.ToLower(kind))
	if err := m.SapOperatorRestClient.Get().Namespace(obj.GetNamespace()).Resource(resource).Name(obj.GetName()).Do(ctx).Into(res); err != nil {
		return err
	}
	if res.GetLabels()[migratedLabel] != "true" {
		return fmt.Errorf("resource already exists and is missing label %v", migratedLabel)
	}
	return nil
}
