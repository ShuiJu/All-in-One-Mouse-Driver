package vaxee

func LoadLegacy(path string) (*Config, error) {
	c, _, err := loadConfig(path)
	return c, err
}
