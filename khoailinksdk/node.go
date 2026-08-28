package khoailinksdk

type Option func(*nodeOptions)
type nodeOptions struct {
	configPath string
}

func Start(ip string, opts ...Option) error {
	options := &nodeOptions{
		configPath: "",
	}

	for _, opt := range opts {
		opt(options)
	}

	khoaiConfig, err := LoadConfig(options.configPath)
	if err != nil {
		return err
	}

	_ = khoaiConfig
	return nil
}
