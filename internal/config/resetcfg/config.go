package resetcfg

import "flag"

type Config struct {
	RootDir string
}

func GetConfig() *Config {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	return cfg
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.RootDir, "d", ".", "project root directory, workdir by default")
}
