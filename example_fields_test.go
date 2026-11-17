package structwalker_test

import (
	"fmt"

	"github.com/rjp2525/structwalker"
)

func ExampleFields() {
	type Database struct {
		Host string `env:"DB_HOST"`
		Port int    `env:"DB_PORT"`
	}
	type Config struct {
		Name     string `env:"APP_NAME"`
		Database Database
	}

	env := map[string]string{"APP_NAME": "billing", "DB_HOST": "db.internal", "DB_PORT": "5432"}

	var cfg Config
	for f := range structwalker.Fields(&cfg) {
		tag, ok := f.Lookup("env")
		if !ok {
			continue
		}
		if err := f.SetValue(env[tag.Name]); err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Printf("%+v\n", cfg)
	// Output:
	// {Name:billing Database:{Host:db.internal Port:5432}}
}
