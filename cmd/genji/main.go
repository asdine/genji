package main

import (
	"log"
	"os"

	"github.com/genjidb/genji/cmd/genji/shell"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Genji"
	app.Usage = "Shell for the Genji database"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "bolt",
			Usage: "use bolt engine",
		},
		&cli.BoolFlag{
			Name:  "badger",
			Usage: "use badger engine",
		},
	}

	app.Action = func(c *cli.Context) error {
		useBolt := c.Bool("bolt")
		useBadger := c.Bool("badger")
		if useBolt && useBadger {
			return cli.NewExitError("cannot use bolt and badger options at the same time", 2)
		}

		dbpath := c.Args().First()

		if (useBolt || useBadger) && dbpath == "" {
			return cli.NewExitError("db path required when using bolt or badger", 2)
		}

		engine := "memory"

		if useBolt || dbpath != "" {
			engine = "bolt"
		}

		if useBadger {
			engine = "badger"
		}

		return shell.Run(&shell.Options{
			Engine: engine,
			DBPath: dbpath,
		})
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
