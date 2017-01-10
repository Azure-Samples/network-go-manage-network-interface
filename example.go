package main

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	westUS          = "westus"
	groupName       = "your-azure-sample-group"
	vNetName        = "vNet"
	nicNameFrontEnd = "nic1"
	nicNameMidTier  = "nic2"
	nicNameBackEnd  = "nic3"
	accountName     = "golangrocksonazure"
	vmName          = "vm"
	vhdURItemplate  = "https://%s.blob.core.windows.net/golangcontainer/%s.vhd"
)

// This example requires that the following environment vars are set:
//
// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID
//

var (
	groupClient      resources.GroupsClient
	vNetClient       network.VirtualNetworksClient
	subnetClient     network.SubnetsClient
	addressClient    network.PublicIPAddressesClient
	interfacesClient network.InterfacesClient
	accountClient    storage.AccountsClient
	vmClient         compute.VirtualMachinesClient
)

func init() {
	subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
	tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(tenantID)
	onErrorFail(err, "OAuthConfigForTenant failed")

	clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
	clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")
	spToken, err := azure.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	onErrorFail(err, "NewServicePrincipalToken failed")

	createClients(subscriptionID, spToken)
}

func main() {
	createResourceGroup()
	createVirtualNetwork()
	subnets := createSubnets()
	pip1 := createPIP("pip1")
	nics := createNICs(subnets, pip1)
	createStorageAccount()
	nirs := buildNIRs(nics)
	createVM(nirs)
	pip2 := createPIP("pip2")
	updateNICwithPIP(nicNameFrontEnd, nics, pip2)
	listNICs()

	fmt.Printf("Press enter to delete NIC '%s'...\n", nicNameMidTier)
	var input string
	fmt.Scanln(&input)

	deleteNIC(nicNameMidTier)
	fmt.Println("Remaining NICs are...")
	listNICs()

	fmt.Print("Press enter to delete all the resources created in this sample...")
	fmt.Scanln(&input)

	deleteResourceGroup()
}

func createResourceGroup() {
	fmt.Println("Create resource group")
	resourceGroup := resources.ResourceGroup{
		Location: to.StringPtr(westUS),
	}
	_, err := groupClient.CreateOrUpdate(groupName, resourceGroup)
	onErrorFail(err, "CreateOrUpdate failed")
}

func createVirtualNetwork() {
	fmt.Println("Create virtual network")
	vNet := network.VirtualNetwork{
		Location: to.StringPtr(westUS),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{"172.16.0.0/16"},
			},
		},
	}
	_, err := vNetClient.CreateOrUpdate(groupName, vNetName, vNet, nil)
	onErrorFail(err, "CreateOrUpdate failed")
}

func createSubnets() *[]network.Subnet {
	fmt.Println("Create subnets")
	subnet := network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{},
	}
	subnetNames := []string{"Front-end", "Mid-tier", "Back-end"}
	subnets := []network.Subnet{}
	for i, n := range subnetNames {
		fmt.Printf("\tCreate subnet: '%s'\n", n)
		subnet.AddressPrefix = to.StringPtr(fmt.Sprintf("172.16.%v.0/24", i+1))
		_, err := subnetClient.CreateOrUpdate(groupName, vNetName, n, subnet, nil)
		onErrorFail(err, "\tCreateOrUpdate failed")

		subnetInfo, err := subnetClient.Get(groupName, vNetName, n, "")
		onErrorFail(err, "\tGet failed")

		subnets = append(subnets, subnetInfo)
	}
	return &subnets
}

// createPIP creates a public IP address
func createPIP(pipName string) *network.PublicIPAddress {
	fmt.Printf("Create public IP address: '%s'\n", pipName)
	pip := network.PublicIPAddress{
		Location: to.StringPtr(westUS),
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			DNSSettings: &network.PublicIPAddressDNSSettings{
				DomainNameLabel: to.StringPtr(fmt.Sprintf("azuresample-%s", pipName)),
			},
		},
	}
	_, err := addressClient.CreateOrUpdate(groupName, pipName, pip, nil)
	onErrorFail(err, "CreateOrUpdate failed")

	fmt.Println("Get public IP address")
	pip, err = addressClient.Get(groupName, pipName, "")
	onErrorFail(err, "Get failed")

	return &pip
}

func createNICs(subnets *[]network.Subnet, pip *network.PublicIPAddress) *[]network.Interface {
	fmt.Println("Create network interfaces (NICs)")
	nic := network.Interface{
		Location: to.StringPtr(westUS),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
					},
				},
			},
		},
	}
	nicNames := []string{
		nicNameFrontEnd,
		nicNameMidTier,
		nicNameBackEnd,
	}
	nics := []network.Interface{}
	for i, n := range nicNames {
		fmt.Printf("\tCreate NIC '%s' using subnet '%s'\n", n, *(*subnets)[i].Name)
		(*nic.IPConfigurations)[0].Name = to.StringPtr(fmt.Sprintf("IPconfig%v", i+1))
		(*nic.IPConfigurations)[0].Subnet = &(*subnets)[i]

		if n == nicNameFrontEnd {
			nic.EnableIPForwarding = to.BoolPtr(true)
			(*nic.IPConfigurations)[0].Primary = to.BoolPtr(true)
			(*nic.IPConfigurations)[0].PublicIPAddress = pip
		} else {
			nic.EnableIPForwarding = nil
			(*nic.IPConfigurations)[0].Primary = nil
			(*nic.IPConfigurations)[0].PublicIPAddress = nil
		}

		_, err := interfacesClient.CreateOrUpdate(groupName, n, nic, nil)
		onErrorFail(err, "CreateOrUpdate failed")

		nicInfo, err := interfacesClient.Get(groupName, n, "")
		onErrorFail(err, "Get failed")

		nics = append(nics, nicInfo)
	}
	return &nics
}

func createStorageAccount() {
	fmt.Println("Create storage account")
	account := storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Name: storage.StandardLRS},
		Location: to.StringPtr(westUS),
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
	}
	_, err := accountClient.Create(groupName, accountName, account, nil)
	onErrorFail(err, "Create failed")
}

func buildNIRs(nics *[]network.Interface) *[]compute.NetworkInterfaceReference {
	fmt.Println("Assign NIC to Network Interface References (NIRs) ")
	nirs := []compute.NetworkInterfaceReference{}
	for i, nic := range *nics {
		fmt.Printf("\tAssign NIC '%s' to NIR %v\n", *nic.Name, i)
		nir := compute.NetworkInterfaceReference{
			ID: nic.ID,
		}
		if *nic.Name == nicNameFrontEnd {
			fmt.Printf("\t%v is assigned to the primary NIR\n", nicNameFrontEnd)
			nir.NetworkInterfaceReferenceProperties = &compute.NetworkInterfaceReferenceProperties{
				Primary: to.BoolPtr(true),
			}
		} else {
			nir.NetworkInterfaceReferenceProperties = &compute.NetworkInterfaceReferenceProperties{
				Primary: to.BoolPtr(false),
			}
		}
		nirs = append(nirs, nir)
	}
	return &nirs
}

func createVM(nirs *[]compute.NetworkInterfaceReference) {
	fmt.Println("Create VM with the assigned NIRs")
	vm := compute.VirtualMachine{
		Location: to.StringPtr(westUS),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.StandardD3V2,
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: to.StringPtr("Canonical"),
					Offer:     to.StringPtr("UbuntuServer"),
					Sku:       to.StringPtr("16.04.0-LTS"),
					Version:   to.StringPtr("latest"),
				},
				OsDisk: &compute.OSDisk{
					Name: to.StringPtr("osDisk"),
					Vhd: &compute.VirtualHardDisk{
						URI: to.StringPtr(fmt.Sprintf(vhdURItemplate, accountName, vmName)),
					},
					CreateOption: compute.FromImage,
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  to.StringPtr(vmName),
				AdminUsername: to.StringPtr("notadmin"),
				AdminPassword: to.StringPtr("Pa$$w0rd1975"),
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{},
			},
		},
	}

	vm.VirtualMachineProperties.NetworkProfile.NetworkInterfaces = nirs

	_, err := vmClient.CreateOrUpdate(groupName, vmName, vm, nil)
	onErrorFail(err, "CreateOrUpdate failed")

}

func updateNICwithPIP(nicName string, nics *[]network.Interface, pip *network.PublicIPAddress) {
	var index int
	for i, nic := range *nics {
		if *nic.Name == nicName {
			index = i
		}
	}
	fmt.Printf("Update NIC '%s' with PIP '%s'\n", nicName, *pip.Name)
	(*(*nics)[index].IPConfigurations)[0].PublicIPAddress = pip
	(*(*nics)[index].IPConfigurations)[0].Primary = to.BoolPtr(true)
	_, err := interfacesClient.CreateOrUpdate(groupName, nicName, (*nics)[index], nil)
	onErrorFail(err, "CreateOrUpdate failed")
}

func listNICs() {
	fmt.Println("Listing NICs")
	list, err := interfacesClient.List(groupName)
	onErrorFail(err, "List failed")
	if list.Value == nil || len(*list.Value) == 0 {
		fmt.Printf("There are no NICs in %s resource group\n", groupName)
	} else {
		for _, nic := range *list.Value {
			printNIC(nic)
		}
	}
}

func deleteNIC(nicName string) {
	fmt.Println("Delete NIC")
	fmt.Println("\tFirst, delete the VM")
	_, err := vmClient.Delete(groupName, vmName, nil)
	onErrorFail(err, "Delete failed")
	fmt.Println("\tSecond, delete the NIC")
	_, err = interfacesClient.Delete(groupName, nicName, nil)
	onErrorFail(err, "Delete failed")
}

func deleteResourceGroup() {
	fmt.Println("Deleting resource group")
	_, err := groupClient.Delete(groupName, nil)
	onErrorFail(err, "Delete failed")
}

// getEnvVarOrExit returns the value of specified environment variable or terminates if it's not defined.
func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		fmt.Printf("Missing environment variable %s\n", varName)
		os.Exit(1)
	}

	return value
}

// onErrorFail prints a failure message and exits the program if err is not nil.
func onErrorFail(err error, message string) {
	if err != nil {
		fmt.Printf("%s: %s\n", message, err)
		os.Exit(1)
	}
}

// printNIC prints basic info about a Network Interface.
func printNIC(nic network.Interface) {
	fmt.Printf("Network interface '%s'\n", *nic.Name)
	fmt.Printf("\tLocation:                    %s\n", *nic.Location)
	fmt.Printf("\tIP forwarding enabled:       %t\n", *nic.EnableIPForwarding)
	fmt.Printf("\tMAC address:                 %s\n", *nic.MacAddress)
	fmt.Printf("\tPrivate IP:                  %s\n", *(*nic.IPConfigurations)[0].PrivateIPAddress)
	fmt.Printf("\tPrivate allocation method:   %s\n", (*nic.IPConfigurations)[0].PrivateIPAllocationMethod)
	fmt.Printf("\tPrimary virtual network ID:  %s\n", *(*nic.IPConfigurations)[0].Subnet.ID)
	fmt.Println()
}

func createClients(subscriptionID string, spToken *azure.ServicePrincipalToken) {
	groupClient = resources.NewGroupsClient(subscriptionID)
	groupClient.Authorizer = spToken

	vNetClient = network.NewVirtualNetworksClient(subscriptionID)
	vNetClient.Authorizer = spToken

	subnetClient = network.NewSubnetsClient(subscriptionID)
	subnetClient.Authorizer = spToken

	addressClient = network.NewPublicIPAddressesClient(subscriptionID)
	addressClient.Authorizer = spToken

	interfacesClient = network.NewInterfacesClient(subscriptionID)
	interfacesClient.Authorizer = spToken

	accountClient = storage.NewAccountsClient(subscriptionID)
	accountClient.Authorizer = spToken

	vmClient = compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = spToken
}
