package shared

type APIKey string

type WithAPIKey struct {
	APIKey APIKey `short:"k" long:"api-key" env:"TINYPNG_API_KEY" description:"TinyPNG API key <https://tinypng.com/dashboard/api>"`
}

func (key APIKey) String() string {
	return string(key)
}

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
