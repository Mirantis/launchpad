package migration

var migrators = make(map[string]func(map[string]interface{}) error)

// Register is used by the migrators to register their migrate function
func Register(apiVersion string, migrator func(map[string]interface{}) error) {
	migrators[apiVersion] = migrator
}

// Migrate will run through the migrations until there is no more migrators found and returns an error if any of the migrations fail
func Migrate(data map[string]interface{}) error {
	for {
		migrator := migrators[data["apiVersion"].(string)]
		if migrator == nil {
			return nil
		}

		if err := migrator(data); err != nil {
			return err
		}
	}
}
