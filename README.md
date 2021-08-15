# Migration to SAP BTP Service Operator 

SAP BTP service operator provides the ability to provision and consume SAP Cloud Platform from Kubernetes cluster in a Kubernetes-native way, based on the K8s Operator pattern.

This document describes the process to migrate a registered Kubernetes platform, based on the Service Catalog (svcat) and Service Manager agent, together with its content, to a SAP BTP service operator-based platform.


## Table of Contents
* [Prerequisites](#prerequisites)
* [Setup](#setup)
* [Migration](#migration)
* [Reference Documentation](#reference-documentation)
* [Support](#support)
* [Troubleshooting](#troubleshooting)
* [Contributions](#contributions)
* [License](#license)

## Prerequisites
- Service Management Control (SMCTL) command line Interface. See [Using the SMCTL](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/0107f3f8c1954a4e96802f556fc807e3.html).
- You must be a SAP BTP subaccount admin

## Setup

1. Obtain the access credentials for the SAP BTP service operator by creating an instance of the SAP Service Manager service (technical name: ```service-manager```) with the ```service-operator-access``` plan and then creating a binding to that instance.</br></br>
   For more information about the process, see the steps 1 and 2 in the **Setup** section of [SAP BTP Service Operator for Kubernetes](https://github.com/SAP/sap-btp-service-operator#setup).</br>
2. Deploy the SAP BTP service operator in the cluster using the access credentials that were obtained in the previous step.</br>**In the ```cluster.id``` parameter, specify the clusterID which is retrieved by running **

```kubectl get configmap -n catalog cluster-info -o yaml```
 ```sh
apiVersion: v1
data:
  __id: ab7fa5e9-5cc3-468f-ab4d-143458785d07__
kind: ConfigMap
metadata:
  creationTimestamp: "2021-08-11T12:29:27Z"
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        .: {}
        f:id: {}
    manager: service-catalog
    operation: Update
    time: "2021-08-11T12:29:27Z"
  name: cluster-info
  namespace: catalog
  resourceVersion: "810930"
  uid: d07d3d11-d5a2-4973-aab0-355cb960ec71
  ```
To delpoy the SAP BTP service operator execute following command: 
   
   ```bash
    helm upgrade --install sap-btp-operator https://github.com/SAP/sap-btp-service-operator/releases/download/<release>/sap-btp-operator-<release>.tgz \
        --create-namespace \
        --namespace=sap-btp-operator \
        --set manager.secret.clientid=<clientid> \
        --set manager.secret.clientsecret=<clientsecret> \
        --set manager.secret.url=<sm_url> \
        --set manager.secret.tokenurl=<url>
        --set cluster.id=<clusterID>
    ```
    
3. Download and install the CLI needed to perform the migration in one of the two following ways:


   * #### Manual installation</br>
     Get the service operator migration CLI:</br>
      ``go get github.com/SAP/sap-btp-service-operator-migration``

     Install the CLI:</br>
     ``go install github.com/SAP/sap-btp-service-operator-migration``

     Rename the CLI binary:</br>
     ``mv $GOPATH/bin/sap-btp-service-operator-migration $GOPATH/bin/migrate``

    * #### Automatic Installation</br>
      You can install the CLI by simply [downloading the latest release](https://github.com/SAP/sap-btp-service-operator-migration/releases).</br>
     
   
 
     #### CLI Overview</br>

     ```
     Migration tool from SVCAT to SAP BTP Service Operator.

     Usage:
       migrate [flags]
       migrate [command]

     Available Commands:
       dry-run     Run migration in dry run mode
       help        Help about any command
       run         Run migration process
       version     Prints migrate version

     Flags:
       -c, --config string       config file (default is $HOME/.migrate/config.json)
       -h, --help                help for migrate
       -k, --kubeconfig string   absolute path to the kubeconfig file (default $HOME/.kube/config)
       -n, --namespace string    namespace to find operator secret (default sap-btp-operator)
     ```

## Migration

1. Prepare your platform for migration by executing the following command: </br>
```smctl curl -X PUT  -d '{"sourcePlatformID": ":platformID"}' /v1/migrate/service_operator/:instanceID``` </br></br>
   Where the parameter values are as following:</br> **platformID** is the ID that was used when [registering the subaccount-scoped Kubernetes platform](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/a55506d6ceea4e3bb4534739bf0699d9.html) </br> **instanceID** is the ID of the ```service-operator-access``` instance created in the step 1 of the [Setup](#setup).</br>
   #### *Note* 
    *Once the migration process has been initiated, the platform is suspended, and you cannot create, update or delete its service instances and service bindings.</br>The process is reversible for as long as you don't start the execution script described in the next step.*
    
   If you'd like to cancel the migration, execute the following command: </br>
```smctl curl -X DELETE  -d '{"sourcePlatformID": ":platformID"}' /v1/migrate/service_operator/:instanceID``` </br></br>
   

2. Execute the migration by running the following command:</br>
  ```migrate run```
  
    #### Migration Script Example
    The script first scans all service instances and service bindings that are managed in your cluster by SVCAT, and verifies whether they are also maintained in SAP BTP.</br>Migration won't be performed on those instances and bindings that aren't found in SAP BTP. Before executing the migration, you can perform a dry run by running the following command:
    
       ```sh
    migrate dry-run
    ```
    To start a migration, run: 
    ```sh
    migrate run
    ![#f03c15]Fetched`#f03c15` 2 instances from SM
    *** Fetched 1 bindings from SM
    *** Fetched 5 svcat instances from cluster
    *** Fetched 2 svcat bindings from cluster
    *** Preparing resources
    
    svcat instance name 'postgres_ins_1' id 'XXX-6134-4c89-bff5-YYY' (postgres_ins_1) not found in SM, skipping it...
    svcat instance name 'xsuaa_ins_2' id 'XXX-cae6-4e23-9e8a-YYY' (xsuaa_ins_2) not found in SM, skipping it...
    svcat instance name 'certificate_3' id 'XXX-dc1d-49d1-86c0-YYY' (certificate_3) not found in SM, skipping it...
    svcat binding name 'auditlog_1' id 'XXX-5226-42cc-81e5-YYY' (auditlog_1) not found in SM, skipping it...
    
    *** found 2 instances and 1 bindings to migrate 
    ```
    
    Before the actual migration starts, the script also validates whether all the resources are migratable.</br> Note that if there is an issue with one or more resources, the process stops.
     ```sh
    svcat instance 'feature-flags-ins3' in namespace 'default' was validated successfully
    svcat instance 'saas-ins1' in namespace 'default' was validated successfully
    svcat binding 'extended-service-bindingn2' in namespace 'default' was validated successfully
    *** Validation completed successfully
    ```
    
    After all of the sources were validated successfully, the actual migration starts.</br>Each service instance and binding is removed from the Service Catalog (SVCAT) and added to the SAP BTP service operator:
    ```
    migrating service instance 'feature-flags-ins' in namespace 'default' (smID: 'XXX-3d1f-40db-8cac-YYY')
    deleting svcat resource type 'serviceinstances' named 'feature-flags-ins' in namespace 'default'
    migrating service instance 'saas-ins1' in namespace 'default' (smID: 'XXX-0f94-4fde-b524-YYY')
    deleting svcat resource type 'serviceinstances' named 'saas-ins1' in namespace 'default'
    migrating service binding 'extended-service-binding2' in namespace 'default' (smID: 'XXX-fc36-4d50-a925-YYY')
    deleting svcat resource type 'servicebindings' named 'extended-service-binding2' in namespace 'default'
    
    *** Migration completed successfully
    ```
    
    #### Dry Run
    Before executing the migration, you can perform a dry run by running the following command:</br>
    ```migrate dry-run```

    The dry run performs both validations mentioned in the previous section, but doesn't perform the actual migration.</br>
    At the end of the run, summary including all encountered errors is shown. 
    
    #### *Note* 
    *Once the migration process has been completed, the SVCAT-based platform is no longer usable.*


## Reference Documentation

## Support
You're welcome to raise issues related to feature requests, bugs, or give us general feedback on this project's GitHub Issues page. 
The SAP BTP service operator project maintainers will respond to the best of their abilities. 

## Troubleshooting

## Contributions


## License
