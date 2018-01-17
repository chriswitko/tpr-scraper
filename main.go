package main

import (
	"distribute"
	"fmt"
	"os"
	"sort"

	"collect"

	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var tasks = []string{"cook", "clean", "laundry", "eat", "sleep", "code"}

func loadEnv() {
	e := godotenv.Load()
	if e != nil {
		panic("Error loading .env file")
	}
}

func main() {
	loadEnv()

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "The Press Review"
	app.Usage = "CLI"
	app.Version = "1.0.0"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Chris Witko",
			Email: "chris.witko@gmail.com",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:     "distribute",
			Category: "Services",
			Aliases:  []string{"d"},
			Usage:    "Deliver newsletters to users",
			Subcommands: []cli.Command{
				{
					Name:  "newsletter",
					Usage: "Deliver newsletter to user",
					Action: func(c *cli.Context) error {
						if c.NumFlags() == 0 {
							return cli.ShowCommandHelp(c, "newsletter")
						}

						// TODO: Connect to db and select users to send to (by default if only one email chris.witko@me.com)
						// TODO: Count number of news ready to be sent
						// TODO: Update user with delivered_at and calculate the next_at
						// TODO: Prepare template and send newsletter

						options := distribute.Options{
							Email: c.String("email"),
							// 	LogMode:    c.Bool("log"),
							// 	TestMode:   c.Bool("test"),
							// 	AllMode:    c.Bool("all"),
							// 	Channels:   c.String("channels"),
							// 	Sections:   c.String("sections"),
							// 	SaveMode:   c.Bool("save"),
							// 	UploadMode: c.Bool("upload"),
							// 	Clusters:   c.Int("upload_clusters"),
							// 	Limit:      c.Int("limit"),
							// 	URL:        c.String("url"),
							// 	Pattern:    c.String("pattern"),
						}
						distribute.SetOptions(options)
						distribute.Execute()
						fmt.Println("SENT!")

						return nil
					},
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "all, a",
							Usage: "Send newsletter to all users",
						},
						cli.StringFlag{
							Name:  "email, e",
							Usage: "Send newsletter to this `EMAIL` (multiple emails separated by comma e.g. email@email1.com,email2@email1.com)",
						},
					},
				},
			},
		},
		{
			Name:     "collect",
			Category: "Services",
			Aliases:  []string{"c"},
			Usage:    "Parse websites for news headlines",
			Action: func(c *cli.Context) error {
				if c.NumFlags() == 0 {
					return cli.ShowCommandHelp(c, "collect")
					// return cli.NewExitError("Some flags are required. Use --help for more info.", 0)
				}

				options := collect.Options{
					LogMode:    c.Bool("log"),
					TestMode:   c.Bool("test"),
					AllMode:    c.Bool("all"),
					Channels:   c.String("channels"),
					Sections:   c.String("sections"),
					SaveMode:   c.Bool("save"),
					UploadMode: c.Bool("upload"),
					Clusters:   c.Int("upload_clusters"),
					Limit:      c.Int("limit"),
					URL:        c.String("url"),
					Pattern:    c.String("pattern"),
				}
				collect.SetOptions(options)
				collect.Execute()

				return nil
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "log, l",
					Usage: "Enable logging",
				},
				cli.BoolFlag{
					Name:  "test, t",
					Usage: "Test mode",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Parse all channels",
				},
				cli.StringFlag{
					Name:  "channels",
					Usage: "Parse only selected channels",
				},
				cli.StringFlag{
					Name:  "sections",
					Usage: "Parse only selected sections",
				},
				cli.BoolFlag{
					Name:  "save, s",
					Usage: "Save all feed to the database",
				},
				cli.BoolFlag{
					Name:  "upload, u",
					Usage: "Upload all images to the CDN",
				},
				cli.IntFlag{
					Name:  "upload_clusters",
					Usage: "Number of workers for upload images to CDN",
					Value: 100,
				},
				cli.IntFlag{
					Name:  "limit",
					Usage: "Number of items per feed",
					Value: 10,
				},
				cli.StringFlag{
					Name:  "url",
					Usage: "URL to a website",
				},
				cli.StringFlag{
					Name:  "pattern",
					Usage: "Pattern to parse a website",
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Run(os.Args)
}
