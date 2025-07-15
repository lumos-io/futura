package providers

type ProviderAuth interface {
	Test() error
}
