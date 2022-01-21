You must create a resource group and service principal.
See https://docs.microsoft.com/en-us/azure/virtual-machines/linux/build-image-with-packer

As an example (replace myResourceGroup and myStorageAccount to your own values)
```
$> az group create -n myResourceGroup -l centralus

$> az account show --query "{ subscription_id: id }"

$> az ad sp create-for-rbac --role Contributor --name myResourceGroup --query "{ client_id: appId, client_secret: password, tenant_id: tenant }"
```

These commands will display the necessary values to prepare the environment as such:

```
export PKR_VAR_az_subscription_id=output from above>
export PKR_VAR_az_client_id=<output from above>
export PKR_VAR_az_client_secret=<output from above>
export PKR_VAR_az_tenant_id=<output from above>

export PKR_VAR_az_image_resource_group=myResourceGroup
export TF_VAR_az_image_resource_group=myResourceGroup
```
