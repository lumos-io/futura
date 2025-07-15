package secrettype

// Cloud constants
const (
	Alibaba      = "alibaba"
	Amazon       = "amazon"
	Azure        = "azure"
	Google       = "google"
	DigitalOcean = "digitalocean"
)

// Amazon keys
const (
	AwsRegion          = "AWS_REGION"
	AwsAccessKeyId     = "AWS_ACCESS_KEY_ID"
	AwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

// Azure keys
const (
	AzureClientID       = "AZURE_CLIENT_ID"
	AzureClientSecret   = "AZURE_CLIENT_SECRET"
	AzureTenantID       = "AZURE_TENANT_ID"
	AzureSubscriptionID = "AZURE_SUBSCRIPTION_ID"
)

// Google keys
const (
	Type          = "type"
	ProjectId     = "project_id"
	PrivateKeyId  = "private_key_id"
	PrivateKey    = "private_key"
	ClientEmail   = "client_email"
	ClientId      = "client_id"
	AuthUri       = "auth_uri"
	TokenUri      = "token_uri"
	AuthX509Url   = "auth_provider_x509_cert_url"
	ClientX509Url = "client_x509_cert_url"
)
