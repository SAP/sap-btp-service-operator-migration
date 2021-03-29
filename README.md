# sap-btp-service-operator-migration

## Prerequisite
1. Service Management Control (SMCTL) Command Line Interface. See [Using the SMCTL](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/0107f3f8c1954a4e96802f556fc807e3.html).
2. Prepare your platform for migration by executing: </br>
```smctl curl -X PUT  -d '{"sourcePlatformID": ":platformID"}' /v1/migrate/service_operator/:instanceID``` </br>
**instanceID**: instance of service-manager/service-operator-access
3. Install [sap btp service operator](https://github.com/SAP/sap-btp-service-operator#setup) by providing parameter ``` --set cluster.id=clusterID ``` the same as of SVCAT 

***Note: you can delete the old platform after successful migration, as it suspended and not usable anymore***

## Getting started

To use the migration CLI you need to download and install it first:

### Approach 1: Manual installation

#### Download CLI
``go get github.com/SAP/sap-btp-service-operator-migration``

#### Install CLI

``go install github.com/SAP/sap-btp-service-operator-migration``

#### Rename the CLI binary

``mv $GOPATH/bin/sap-btp-service-operator-migration $GOPATH/bin/migrate``

### Approach 2: Get the latest CLI release
You can get started with the CLI by simply downloading the latest release from [HERE](https://github.com/SAP/sap-btp-service-operator-migration/releases).


## Using CLI

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

## Example usage of CLI:

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
*** Migration completed successfull

```

