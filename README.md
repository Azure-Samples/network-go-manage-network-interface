---
services: Network
platforms: Go
author: mcardosos
---

# Getting Started with Network - Manage Network Interface - in Go

      Azure Network sample for managing network interfaces -
       - Create a virtual machine with multiple network interfaces
       - Configure a network interface
       - List network interfaces
       - Delete a network interface.


## Running this Sample

1. If you don't already have it, [install Go](https://golang.org/dl/).

1. Clone the repository.

```
git clone https://github.com:Azure-Samples/network-go-manage-network-interface.git
```

1. Install the dependencies using glide.

```
cd network-go-manage-network-interface
glide install
```

1. Create an Azure service principal either through
    [Azure CLI](https://azure.microsoft.com/documentation/articles/resource-group-authenticate-service-principal-cli/),
    [PowerShell](https://azure.microsoft.com/documentation/articles/resource-group-authenticate-service-principal/)
    or [the portal](https://azure.microsoft.com/documentation/articles/resource-group-create-service-principal-portal/).

1. Set the following environment variables using the information from the service principle that you created.

```
export AZURE_TENANT_ID={your tenant id}
export AZURE_CLIENT_ID={your client id}
export AZURE_CLIENT_SECRET={your client secret}
export AZURE_SUBSCRIPTION_ID={your subscription id}
```

    > [AZURE.NOTE] On Windows, use `set` instead of `export`.

1. Run the sample.

```
go run example.go
```

## More information

Please refer to [Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go) for more information.
If you don't have a Microsoft Azure subscription you can get a FREE trial account [here](http://go.microsoft.com/fwlink/?LinkId=330212)

---

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.