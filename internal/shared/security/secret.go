package security

const (
	DefaultSecretLength  = 32
	DefaultSecretCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)

type Secret struct {
	generator *Generator
}

func NewSecret() *Secret {
	return &Secret{
		generator: NewGenerator(),
	}
}

// GenerateKey generates a URL-safe base64 encoded secret key
func (s *Secret) Generate(length int) (string, error) {
	if length < 0 {
		return "", ErrSecretLength
	}
	if length == 0 {
		length = DefaultSecretLength
	}
	return s.generator.Generate(length, DefaultSecretCharset)
}
