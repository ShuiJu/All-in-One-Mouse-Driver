package lamzu

// LoadLegacy reads an existing configuration without changing the source file.
func LoadLegacy(path string) (*Config, error) {
	c, _, err := loadConfig(path)
	return c, err
}
