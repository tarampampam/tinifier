package shared

// APIKey is API key.
type APIKey string

// WithAPIKey is used for including API key property into different structs.
type WithAPIKey struct {
	APIKey APIKey `short:"k" long:"api-key" env:"TINYPNG_API_KEY" required:"true" description:"TinyPNG API key <https://tinypng.com/dashboard/api>"` //nolint:lll
}

func (key APIKey) String() string {
	return string(key)
}

// Masked returns API key with replacing chars in a middle with asterisk.
func (key APIKey) Masked() string {
	const offsets int = 4

	var (
		rs = []rune(key)
		l  = len(rs)
	)

	for i := offsets; i < l-offsets; i++ {
		rs[i] = '*'
	}

	return string(rs)
}
