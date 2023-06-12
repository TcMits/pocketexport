package main

import (
	"log"

	"github.com/TcMits/pocketexport"
	"github.com/pocketbase/pocketbase"
	// "github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	app := pocketbase.New()

	// migratecmd.MustRegister(app, app.RootCmd, &migratecmd.Options{
	// 	Automigrate: true, // auto creates migration files when making collection changes
	// })

	// register pocketexport app
	if err := pocketexport.Register(app, pocketexport.GenerateInBackground(true)); err != nil {
		log.Fatal(err)
	}

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
