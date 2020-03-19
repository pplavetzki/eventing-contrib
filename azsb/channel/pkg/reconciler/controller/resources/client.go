package resources

import (
	"os"

	azsbus "github.com/Azure/azure-service-bus-go"
)

var (
	tenantID      = ""
	clientID      = ""
	secret        = "d1f6672c-8cce-4b83-b3fc-c4e4598c2236"
	subscription  = ""
	resourceGroup = ""
	location      = ""
	connection    = ""
	namespace     = ""
	host          = ""
)

// MakeClient returns the Azure Service Bus Client
func MakeClient(opts ...azsbus.NamespaceOption) (*azsbus.Namespace, error) {
	return azsbus.NewNamespace(append(opts, azsbus.NamespaceWithConnectionString(connection))...)
}

func init() {
	tenantID = os.Getenv("TENANT")
	clientID = os.Getenv("SP_ID")
	secret = os.Getenv("SP_PASSWORD")
	subscription = os.Getenv("SUBSCRIPTION")
	resourceGroup = os.Getenv("RESOURCE_GROUP")
	location = os.Getenv("REGION")
	connection = os.Getenv("SB_CONNECTION")
}
