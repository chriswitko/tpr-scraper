package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/urfave/cli"
)

func main() {
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
	app.Copyright = "(c) 2017 Chris Witko, SUBURB STUDIO"

	app.Commands = []cli.Command{
		{
			Name:     "postman",
			Category: "Services",
			Aliases:  []string{"d"},
			Usage:    "Deliver newsletters to users",
			Subcommands: []cli.Command{
				{
					Name:  "deliver",
					Usage: "Deliver newsletter to user",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "all, a",
							Usage: "All users",
						},
						cli.StringFlag{
							Name:  "email, e",
							Usage: "Send to this `EMAIL`",
						},
					},
				},
			},
			// Action: func(c *cli.Context) error {
			// fmt.Println("completed task: ", c.Args().First())

			// data := [][]string{
			// 	[]string{"1/1/2014", "Domain name", "2233", "$10.98"},
			// 	[]string{"1/1/2014", "January Hosting", "2233", "$54.95"},
			// 	[]string{"1/4/2014", "February Hosting", "2233", "$51.00"},
			// 	[]string{"1/4/2014", "February Extra Bandwidth", "2233", "$30.00"},
			// }

			// table := tablewriter.NewWriter(os.Stdout)
			// table.SetHeader([]string{"Date", "Description", "CV2", "Amount"})
			// table.SetFooter([]string{"", "", "Total", "$146.93"}) // Add Footer
			// table.SetBorder(false)                                // Set Border to false

			// table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor},
			// 	tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			// 	tablewriter.Colors{tablewriter.BgRedColor, tablewriter.FgWhiteColor},
			// 	tablewriter.Colors{tablewriter.BgCyanColor, tablewriter.FgWhiteColor})

			// table.SetColumnColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
			// 	tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			// 	tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
			// 	tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor})

			// table.SetFooterColor(tablewriter.Colors{}, tablewriter.Colors{},
			// 	tablewriter.Colors{tablewriter.Bold},
			// 	tablewriter.Colors{tablewriter.FgHiRedColor})

			// table.AppendBulk(data)
			// table.Render()

			// return nil
			// },
		},
		{
			Name:     "test",
			Category: "Testing actions",
			Aliases:  []string{"t"},
			Usage:    "Test patterns on the website",
			Action: func(c *cli.Context) error {
				fmt.Println("completed task: ", c.Args().First())
				return nil
			},
		},
		{
			Name:     "parser",
			Category: "Services",
			Aliases:  []string{"p"},
			Usage:    "Parse websites for news headlines",
			Action: func(c *cli.Context) error {
				fmt.Println("completed task: ", c.Args().First())
				return nil
			},
		},
		{
			Name:     "channel",
			Category: "Models",
			Aliases:  []string{"c"},
			Usage:    "Manage channels",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "Add a new channel",
					Action: func(c *cli.Context) error {
						fmt.Println("new task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "update",
					Usage: "Update an existing channel",
					Action: func(c *cli.Context) error {
						fmt.Println("updated channel args: ", c.Args().First())
						fmt.Println("updated channel flags: ", c.String("code"))
						return nil
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "code, c",
							Usage: "Set/Update code",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "Set/Update name",
						},
						cli.StringFlag{
							Name:  "url, u",
							Usage: "Set/Update URL",
						},
						cli.StringFlag{
							Name:  "icon, i",
							Usage: "Set/Update icon",
						},
					},
				},
				{
					Name:  "enable",
					Usage: "Enable an existing channel",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "disable",
					Usage: "Disable an existing channel",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "remove",
					Usage: "Remove an existing channel",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
			},
		},
		{
			Name:     "section",
			Category: "Models",
			Aliases:  []string{"s"},
			Usage:    "Manage sections",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "Add a new section",
					Action: func(c *cli.Context) error {
						fmt.Println("new task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "update",
					Usage: "Update an existing section",
					Action: func(c *cli.Context) error {
						fmt.Println("updated channel args: ", c.Args().First())
						fmt.Println("updated channel flags: ", c.String("config"))
						return nil
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "config, c",
							Usage: "Load configuration from `FILE`",
						},
					},
				},
				{
					Name:  "enable",
					Usage: "Enable an existing section",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "disable",
					Usage: "Disable an existing section",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "remove",
					Usage: "Remove an existing section",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Run(os.Args)
}
