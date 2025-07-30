package storage

type clickhouseStorage struct{}

func NewClickhouseStorage() (Storage, error) {
	return &clickhouseStorage{}, nil
}
