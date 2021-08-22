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

1. Obtain the access credentials for the SAP BTP service operator by creating an instance of the SAP Service Manager (technical name: ```service-manager```) with the ```service-operator-access``` plan and then creating a binding to that instance.</br></br>
   For more information about the process, see the steps 1 and 2 in the **Setup** section of [SAP BTP Service Operator for Kubernetes](https://github.com/SAP/sap-btp-service-operator#setup).</br><br>
2. Deploy the SAP BTP service operator in the cluster by using the access credentials that were obtained in the previous step.<br><br>*Note*<br>*The cluster needs to be the same cluster with your svcat-based content, therefore you'll need to specify the cluster ID to identify it.*</br>*To find the cluster.id value that you'll use in the deployment script, run the following command:*

   *```kubectl get configmap -n catalog cluster-info -o yaml``` and search for the **id** value in the output.*

Output example:

 ```sh
apiVersion: v1
data:
  **id: ab7fa5e9-5cc3-468f-ab4d-143458785d07**
kind: ConfigMap
metadata:
 .
 .
  ```
To delpoy the SAP BTP service operator, execute the following command: 
   
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
  
  
2. At this point, you have two options at your disposal:<br>
   - Execute the actual migration by running the following command: ```migrate run```.
   - Perform a dry run before you execute the migration by running: ```migrate dry-run```.
  
   Dry run is useful if you wish to execute the scan and validation steps described in the migration script example below without performing the actual migration.<br>At the end of the run, summary including all encountered errors is shown.<br>This way, you can decide whether to continue with the migration or first fix the issues.
   
    #### *Note* 
    *Once the actual migration process has been initiated, the platform is suspended, and you cannot create, update, or delete its service instances and service bindings.</br>The process is reversible for as long as the actual migration of the resources does not start (described below in the part 3 of the migration script example).*
    
   *To cancel the migration, execute the following command: </br>
```smctl curl -X DELETE  -d '{"sourcePlatformID": ":platformID"}' /v1/migrate/service_operator/:instanceID```* </br></br>
   
#### Migration Script Example
   
   1. The script first scans all service instances and service bindings that are managed in your cluster by SVCAT, and verifies whether they are also maintained in SAP BTP.</br>Migration won't be performed on those instances and bindings that aren't found in SAP BTP:

  ```sh
    migrate run
    
    Fetched 2 instances from SM
    *** Fetched 1 bindings from SM
    *** Fetched 5 svcat instances from cluster
    *** Fetched 2 svcat bindings from cluster
    *** Preparing resources
    
    svcat instance name 'some_instance_name_1' id 'XXX-6134-4c89-bff5-YYY' (some_instance_name_1) not found in SM, skipping it...
    svcat instance name 'some_instance_name_2' id 'XXX-cae6-4e23-9e8a-YYY' (some_instance_name_2) not found in SM, skipping it...
    svcat instance name 'some_instance_name_3' id 'XXX-dc1d-49d1-86c0-YYY' (some_instance_name_3) not found in SM, skipping it...
    svcat binding name 'some_binding_name_1' id 'XXX-5226-42cc-81e5-YYY' (some_binding_name_1) not found in SM, skipping it...
    
    *** found 2 instances and 1 bindings to migrate 
  ```
  2. Before the actual migration starts, the script also validates whether all the resources are migratable.</br> Note that if there is an issue with one or more resources, the process stops.
  ```html
    svcat instance 'some_instance_name_4' in namespace 'namespace_name' was validated successfully
    svcat instance 'some_instance_name_5' in namespace 'namespace_name' was validated successfully
    svcat binding 'some_binding_name_2' in namespace 'namespace_name' was validated successfully
    
    *** Validation completed successfully
   ```
    
  3. After all of the sources were validated successfully, the actual migration starts.</br>Each service instance and binding is removed from the Service Catalog (SVCAT) and added to the SAP BTP service operator:
  ```
    migrating service instance 'some_instance_name_4' in namespace 'namespace_name' (smID: 'XXX-3d1f-40db-8cac-YYY')
    deleting svcat resource type 'serviceinstances' named 'some_instance_name_4' in namespace 'namespace_name'
    migrating service instance 'some_instance_name_5' in namespace 'namespace_name' (smID: 'XXX-0f94-4fde-b524-YYY')
    deleting svcat resource type 'serviceinstances' named 'some_instance_name_5' in namespace 'namespace_name'
    migrating service binding 'some_binding_name_2' in namespace 'namespace_name' (smID: 'XXX-fc36-4d50-a925-YYY')
    deleting svcat resource type 'servicebindings' named 'some_binding_name_2' in namespace 'namespace_name'
    
    *** Migration completed successfully
  ```
   
    
  #### *Note* 
   *Once the migration process has been completed, the SVCAT-based platform is no longer usable.* 
   
## Reference Documentation

## Support
You're welcome to raise issues related to feature requests, bugs, or give us general feedback on this project's GitHub Issues page. 
The SAP BTP service operator project maintainers will respond to the best of their abilities. 

## Troubleshooting

## Contributions


## License
