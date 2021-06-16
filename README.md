# Migration to SAP BTP Service Operator 

SAP BTP service operator provides the ability to provision and consume SAP Cloud Platform from Kubernetes cluster in a Kubernetes-native way, based on the K8s Operator pattern.

Our mission was to define and design a solution that would make it possible to migrate a registered Kubernetes platform, based on the Service Catalog (svcat) and Service Manager agent, together with its content, to s SAP BTP service operator-based platform.

## Table of Contents
* [Prerequisites](#prerequisites)
* [Setup](#setup)
* [Preparing the Migration](#preparing-the-migration)
* [Executing the Migration](#executing-the-migration)
* [Using the CLI (Example)](#using-the-cli-example)
* [Reference Documentation](#reference-documentation)
* [Support](#support)
* [Troubleshooting](#troubleshooting)
* [Contributions](#contributions)
* [License](#license)

## Prerequisites
- Service Management Control (SMCTL) command line Interface. See [Using the SMCTL](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/0107f3f8c1954a4e96802f556fc807e3.html).
- You must be a SAP BTP subaccount admin


## Setup


1. Obtain the access credentials for the SAP BTP service operator by creating an instance of the SAP Service Manager service (technical name: ```service-manager```) with the ```service-operator-access``` plan and then creating a binding to that instance.</br>
   For more information about the process, see the steps 1 and 2 in the **Setup** section of [SAP BTP Service Operator for Kubernetes](https://github.com/SAP/sap-btp-service-operator#setup).</br>
2. Deploy the SAP BTP service operator in the cluster using the access credentials that were obtained in the previous step.</br>**In the ```cluster.id``` parameter, specify the clusterID that was used when [registering the subaccount-scoped Kubernetes platform](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/a55506d6ceea4e3bb4534739bf0699d9.html) that you want to migrate.**
   
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

## Preparing the Migration

1. Prepare your platform for migration by executing the following command: </br>
```smctl curl -X PUT  -d '{"sourcePlatformID": ":platformID"}' /v1/migrate/service_operator/:instanceID``` </br>
   Where:</br> **platformID** is the ID that was used when [registering the subaccount-Kubernetes platform](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/a55506d6ceea4e3bb4534739bf0699d9.html) </br> **instanceID** is the ID of the ```service-operator-access``` instance created in the step 1 of the [Setup](#setup).</br></br>
   #### *Note* 
   
   *Once the migration has been initiated, the platform is suspended, and you cannot create, update or delete its service instances and service bindings.*
   
## Executing the Migration

Run the migration script by executing the following command:</br>
```migrate run```

The script scans all service instances and service bindings that are managed in your cluster by SVCAT, and verifies whether they are also maintained in SAP BTP.</br>Migration   won't be performed on those instances and bindings that aren't found in SAP BTP.</br></br>
Before the actual migration starts, the script also validates whether all the resources are migratable. [what does this mean] If there is an issue with one or more resources,   the process stops.</br></br>
After all of the sources were validated successfully, the actual migration starts. Each service instance and binding is removed from the Service Catalog (SVCAT) and added to     the SAP BTP service operator.</br></br>
## Dry Run
Before executant the migration, you can perform a dry run by executing the following command:</br>
```migrate dry-run```

The dry run performs both validations mentioned in the previous section, but doesn't perform the migration.</br>
At the end of the run, summary including all encountered errors is shown. 


## Using the CLI (Example):

```sh
# Run migration including pre migration validations
migrate run
*** Fetched 2 instances from SM
*** Fetched 1 bindings from SM
*** Fetched 5 svcat instances from cluster
*** Fetched 2 svcat bindings from cluster
*** Preparing resources
svcat instance name 'test11' id 'XXX-6134-4c89-bff5-YYY' (test11) not found in SM, skipping it...
svcat instance name 'test21' id 'XXX-cae6-4e23-9e8a-YYY' (test21) not found in SM, skipping it...
svcat instance name 'test22' id 'XXX-dc1d-49d1-86c0-YYY' (test22) not found in SM, skipping it...
svcat binding name 'test5' id 'XXX-5226-42cc-81e5-YYY' (test5) not found in SM, skipping it...
*** found 2 instances and 1 bindings to migrate
*** Validating
svcat instance 'test32' in namespace 'default' was validated successfully
svcat instance 'test35' in namespace 'default' was validated successfully
svcat binding 'test31' in namespace 'default' was validated successfully
*** Validation completed successfully
migrating service instance 'test32' in namespace 'default' (smID: 'XXX-3d1f-40db-8cac-YYY')
deleting svcat resource type 'serviceinstances' named 'test32' in namespace 'default'
migrating service instance 'test35' in namespace 'default' (smID: 'XXX-0f94-4fde-b524-YYY')
deleting svcat resource type 'serviceinstances' named 'test35' in namespace 'default'
migrating service binding 'test31' in namespace 'default' (smID: 'XXX-fc36-4d50-a925-YYY')
deleting svcat resource type 'servicebindings' named 'test31' in namespace 'default'
*** Migration completed successfully

```
## Reference Documentation

## Support
You're welcome to raise issues related to feature requests, bugs, or give us general feedback on this project's GitHub Issues page. 
The SAP BTP service operator project maintainers will respond to the best of their abilities. 

## Troubleshooting

## Contributions

## License
