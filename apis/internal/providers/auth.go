package providers

type ProviderAuth interface {
	Test() error
	GetCredentials(organizationID string, secretID string) (map[string]string, error)
	SetCredentials(organizationID, secretID string, creds map[string]string) error
	UpdateCredentials(organizationID, secretID string, creds map[string]string) error
	DeleteCredentials(organizationID, secretID string) error
}
