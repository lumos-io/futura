package aws

type AWSProvider struct {
	account  string
	rolename string
}

func New(account, role string) (*AWSProvider, error) {
	return &AWSProvider{
		account:  account,
		rolename: role,
	}, nil
}
