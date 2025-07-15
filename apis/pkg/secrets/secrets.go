package secrets

type SecretStorage interface {
	Get(key string) (string, error)
	Set(key string, value string) error
}
