package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

const (
	ProviderGCP   = "gcp"
	ProviderAWS   = "aws"
	ProviderAzure = "azure"
)

type (
	Topology struct {
		data  TopologyData
		image []byte
		Name  string
	}

	TopoKey struct {
		Kind     string
		Type     string
		Provider string
	}

	TopologyData struct {
		IconData []TopologyIconData `json:"topologyIconData"`
		EdgeData []TopologyEdgeData `json:"topologyEdgeData"`
	}

	TopologyIconData struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Image string `json:"image"`
		Kind  string `json:"kind"`
		Type  string `json:"type"`
	}

	TopologyEdgeData struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}

	TopoMap map[TopoKey]string
)

// DiagramEntityToImgPath values are relative to the base URL: https://github.com/mingrammer/diagrams/tree/master/resources
var DiagramEntityToImgPath = TopoMap{
	{}: "generic/blank/blank.png",

	{Kind: GatewayKind}:                     "generic/network/subnet.png",
	{Kind: ExecutionUnitKind}:               "generic/compute/rack.png",
	{Kind: string(PersistKVKind)}:           "generic/storage/storage.png",
	{Kind: string(PersistFileKind)}:         "generic/storage/storage.png",
	{Kind: string(PersistSecretKind)}:       "generic/storage/storage.png",
	{Kind: string(PersistORMKind)}:          "generic/database/sql.png",
	{Kind: string(PersistRedisNodeKind)}:    "generic/storage/storage.png",
	{Kind: string(PersistRedisClusterKind)}: "generic/storage/storage.png",
	{Kind: PubSubKind}:                      "generic/blank/blank.png",

	// Use AWS as the ultimate fallback for the Kind, so don't specify Provider.
	{Kind: GatewayKind, Provider: ProviderAWS}:                          "aws/network/api-gateway.png",
	{Kind: ExecutionUnitKind, Provider: ProviderAWS}:                    "aws/compute/lambda.png",
	{Kind: string(PersistKVKind), Provider: ProviderAWS}:                "aws/database/dynamodb.png",
	{Kind: string(PersistFileKind), Provider: ProviderAWS}:              "aws/compute/simple-storage-service-s3.png",
	{Kind: string(PersistSecretKind), Provider: ProviderAWS}:            "aws/security/secrets-manager.png",
	{Kind: string(PersistORMKind), Provider: ProviderAWS}:               "aws/database/rds.png",
	{Kind: string(PersistRedisNodeKind), Provider: ProviderAWS}:         "aws/database/elasticache-for-redis.png",
	{Kind: string(PersistRedisClusterKind), Provider: ProviderAWS}:      "aws/database/elasticache-for-redis.png", // Theres no memoryDB at the moment
	{Kind: PubSubKind, Provider: ProviderAWS}:                           "aws/integration/simple-notification-service-sns.png",
	{Kind: NetworkLoadBalancerKind, Provider: ProviderAWS}:              "aws/network/elb-network-load-balancer.png",
	{Kind: ExecutionUnitKind, Type: "ecs", Provider: ProviderAWS}:       "aws/compute/fargate.png",
	{Kind: ExecutionUnitKind, Type: "eks", Provider: ProviderAWS}:       "aws/compute/elastic-kubernetes-service.png",
	{Kind: ExecutionUnitKind, Type: "apprunner", Provider: ProviderAWS}: "aws/compute/app-runner.png",

	{Kind: GatewayKind, Provider: ProviderGCP}:               "gcp/network/api-gateway.png",
	{Kind: ExecutionUnitKind, Provider: ProviderGCP}:         "gcp/compute/run.png",
	{Kind: string(PersistKVKind), Provider: ProviderGCP}:     "gcp/database/firestore.png",
	{Kind: string(PersistFileKind), Provider: ProviderGCP}:   "gcp/storage/filestore.png",
	{Kind: string(PersistSecretKind), Provider: ProviderGCP}: "gcp/security/resource-manager.png",
	{Kind: string(PersistORMKind), Provider: ProviderGCP}:    "gcp/database/sql.png",
	{Kind: PubSubKind, Provider: ProviderGCP}:                "gcp/analytics/pubsub.png",

	{Kind: GatewayKind, Provider: ProviderAzure}:               "azure/network/application-gateway.png",
	{Kind: ExecutionUnitKind, Provider: ProviderAzure}:         "azure/compute/function-apps.png",
	{Kind: string(PersistKVKind), Provider: ProviderAzure}:     "azure/database/cosmos-db.png",
	{Kind: string(PersistFileKind), Provider: ProviderAzure}:   "azure/storage/blob-storage.png",
	{Kind: string(PersistSecretKind), Provider: ProviderAzure}: "azure/security/key-vaults.png",
	{Kind: string(PersistORMKind), Provider: ProviderAzure}:    "azure/database/sql-databases.png",
	{Kind: PubSubKind, Provider: ProviderAzure}:                "azure/analytics/event-hubs.png",
}

// DiagramEntityToCode values are modules from https://github.com/CloudCompilers/topology-visualizer/blob/main/app/app.py and
// the class from https://github.com/mingrammer/diagrams/tree/master/diagrams
var DiagramEntityToCode = TopoMap{
	{}: "",

	{Kind: GatewayKind}:                     `generic_network.Subnet("%s")`,
	{Kind: ExecutionUnitKind}:               `generic_compute.Rack("%s")`,
	{Kind: string(PersistKVKind)}:           `generic_storage.Storage("%s")`,
	{Kind: string(PersistFileKind)}:         `generic_storage.Storage("%s")`,
	{Kind: string(PersistSecretKind)}:       `generic_storage.Storage("%s")`,
	{Kind: string(PersistORMKind)}:          `generic_database.Sql("%s")`,
	{Kind: string(PersistRedisNodeKind)}:    `generic_storage.Storage("%s")`,
	{Kind: string(PersistRedisClusterKind)}: `generic_storage.Storage("%s")`,
	{Kind: PubSubKind}:                      `generic_blank.Blank("%s")`,

	// Use AWS as the ultimate fallback for the Kind, so don't specify Provider.
	{Kind: GatewayKind, Provider: ProviderAWS}:                     `aws_network.APIGateway("%s")`,
	{Kind: ExecutionUnitKind, Provider: ProviderAWS}:               `aws_compute.Lambda("%s")`,
	{Kind: string(PersistKVKind), Provider: ProviderAWS}:           `aws_database.Dynamodb("%s")`,
	{Kind: string(PersistFileKind), Provider: ProviderAWS}:         `aws_storage.S3("%s")`,
	{Kind: string(PersistSecretKind), Provider: ProviderAWS}:       `aws_security.SecretsManager("%s")`,
	{Kind: string(PersistORMKind), Provider: ProviderAWS}:          `aws_database.RDS("%s")`,
	{Kind: string(PersistRedisNodeKind), Provider: ProviderAWS}:    `aws_database.ElasticacheForRedis("%s")`,
	{Kind: string(PersistRedisClusterKind), Provider: ProviderAWS}: `aws_database.ElasticacheForRedis("%s")`, // Theres no memoryDB at the moment
	{Kind: string(PubSubKind), Provider: ProviderAWS}:              `aws_integration.SNS("%s")`,

	{Kind: ExecutionUnitKind, Type: "ecs", Provider: ProviderAWS}:       `aws_compute.Fargate("%s")`,
	{Kind: ExecutionUnitKind, Type: "eks", Provider: ProviderAWS}:       `aws_compute.EKS("%s")`,
	{Kind: ExecutionUnitKind, Type: "apprunner", Provider: ProviderAWS}: `generic_compute.Rack("AR: %s")`, // Using generic until mingrammer cuts a release with AppRunner
	{Kind: NetworkLoadBalancerKind, Provider: ProviderAWS}:              `aws_network.ElbNetworkLoadBalancer("%s")`,

	{Kind: GatewayKind, Provider: ProviderGCP}:               `custom.Custom("%s", "./images/gcp-api-gateway.png")`,
	{Kind: string(PersistKVKind), Provider: ProviderGCP}:     `gcp_database.Firestore("%s")`,
	{Kind: ExecutionUnitKind, Provider: ProviderGCP}:         `gcp_compute.Run("%s")`,
	{Kind: string(PersistFileKind), Provider: ProviderGCP}:   `gcp_storage.Filestore("%s")`,
	{Kind: string(PersistSecretKind), Provider: ProviderGCP}: `gcp_security.ResourceManager("%s")`,
	{Kind: string(PersistORMKind), Provider: ProviderGCP}:    `gcp_database.SQL("%s")`,
	{Kind: PubSubKind, Provider: ProviderGCP}:                `gcp_analytics.Pubsub("%s")`,

	{Kind: GatewayKind, Provider: ProviderAzure}:               `azure_network.ApplicationGateway("%s")`,
	{Kind: string(PersistKVKind), Provider: ProviderAzure}:     `azure_database.CosmosDb("%s")`,
	{Kind: ExecutionUnitKind, Provider: ProviderAzure}:         `azure_compute.FunctionApps("%s")`,
	{Kind: string(PersistFileKind), Provider: ProviderAzure}:   `azure_storage.BlobStorage("%s")`,
	{Kind: string(PersistSecretKind), Provider: ProviderAzure}: `azure_security.KeyVaults("%s")`,
	{Kind: string(PersistORMKind), Provider: ProviderAzure}:    `azure_database.SQLDatabases("%s")`,
	{Kind: PubSubKind, Provider: ProviderAzure}:                `azure_analytics.EventHubs("%s")`,
}

var TopologyKind = "topology"

func NewTopology(name string, data TopologyData, image []byte) *Topology {
	return &Topology{
		Name:  name,
		data:  data,
		image: image,
	}
}

func (Topology) Type() string { return "" }

func (t *Topology) Key() ResourceKey {
	return ResourceKey{
		Name: t.Name,
		Kind: TopologyKind,
	}
}

func (t *Topology) OutputTo(dest string) error {
	jsonP := path.Join(dest, fmt.Sprintf("%s.json", t.Name))
	imageP := path.Join(dest, fmt.Sprintf("%s.png", t.Name))

	// write the json
	file, err := os.OpenFile(jsonP, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")

	err = enc.Encode(t.data)
	if err != nil {
		return err
	}

	// write the image
	err = os.WriteFile(imageP, t.image, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (t *Topology) GetTopologyData() TopologyData {
	return t.data
}

// Get returns an exact match for `k` if present, otherwise fallback to `resType = ""` then further to `provider = ""`. Finally,
// if no matches, uses the default/blank of `kind = ""` (all fields empty). Returns the value and the Key which resolved to that value.
func (m TopoMap) Get(kind string, resType string, provider string) (string, TopoKey) {
	k := TopoKey{kind, resType, provider}
	if v, ok := m[k]; ok {
		return v, k
	}
	k.Type = ""
	if v, ok := m[k]; ok {
		return v, k
	}
	k.Provider = ""
	if v, ok := m[k]; ok {
		return v, k
	}
	k = TopoKey{}
	return m[k], k
}
