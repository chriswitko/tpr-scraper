package distribute

import (
	"database"
	"fmt"
	"jaro"
	"log"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"util"

	"github.com/chriswitko/tprses"

	"github.com/araddon/dateparse"
	"github.com/fatih/structs"
	"github.com/imdario/mergo"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	version string = "master"
)

var debug bool

// User type
type User struct {
	ID             bson.ObjectId `structs:"id" json:"id,omitempty" bson:"_id"`
	Role           string        `structs:"role" form:"role" json:"role" bson:"role"`
	Email          string        `structs:"email" json:"email"`
	Icon           string        `structs:"icon" json:"icon"`
	Timezone       string        `structs:"timezone" json:"timezone,omitempty" bson:"timezone"`
	TimeFormat     string        `structs:"time_format" json:"time_format,omitempty" bson:"time_format"`
	Language       string        `structs:"language" json:"language,omitempty" bson:"language"`
	Fullname       string        `structs:"fullname" json:"fullname,omitempty" bson:"fullname"`
	Hours          []string      `structs:"hours" json:"hours,omitempty" bson:"hours"`
	Days           []int         `structs:"days" json:"days,omitempty" bson:"days"`
	Channels       []string      `structs:"channels" json:"channels,omitempty" bson:"channels"`
	Topics         []string      `structs:"topics" json:"topics,omitempty" bson:"topics"`
	CreatedAt      time.Time     `structs:"created_at" json:"created_at" bson:"created_at"`
	SubscribedAt   time.Time     `structs:"subscribed_at" json:"subscribed_at,omitempty" bson:"subscribed_at"`
	UnsubscribedAt time.Time     `structs:"unsubscribed_at" json:"unsubscribed_at,omitempty" bson:"unsubscribed_at"`
	NextAt         time.Time     `structs:"next_at" json:"next_at,omitempty" bson:"next_at"`
	DeliveredAt    time.Time     `structs:"delivered_at" json:"delivered_at,omitempty" bson:"delivered_at"`
	LastSignInAt   time.Time     `structs:"last_signin_at" json:"last_signin_at,omitempty" bson:"last_signin_at"`
}

// News is part of feeditem
type News struct {
	Title            string    `bson:"title"`
	Description      string    `bson:"description"`
	Link             string    `bson:"url"`
	Channel          string    `bson:"channel"`
	Section          string    `bson:"section"`
	CreatedAt        time.Time `bson:"created_at"`
	Hash             string    `bson:"hash"`
	Position         int       `bson:"position_idx"`
	CanonicalURL     string    `bson:"canonical_url"`
	AmpURL           string    `bson:"amp_url"`
	OriginalImageURL string    `bson:"original_image_url"`
	ImageUUID        string    `bson:"image_uuid"`
	ImageWidth       int       `bson:"image_width"`
	ImageHeight      int       `bson:"image_height"`
	History          []int     `bson:"history_idx"`
	Hostname         string    `structs:"hostname" json:"hostname" bson:"hostname"`
}

// Channel is part of feeditem
type Channel struct {
	Name string `bson:"name"`
	Code string `bson:"code"`
}

// Newspaper is a collection of news
type Newspaper []News

// MongoConnection for all endpoints
var mu = &sync.Mutex{}

// Options - a global settings
type Options struct {
	Email      string
	AllMode    bool
	LogMode    bool
	MemoryMode bool
}

var globalOptions Options
var db database.MongoConnection
var newspaper Newspaper

// SetOptions manage global options
func SetOptions(options Options) {
	mergo.Merge(&globalOptions, options)
}

func logAllocMemory() {
	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		log.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC = %v\n\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
		time.Sleep(1 * time.Second)
	}
}

func catchPanic(err *error, functionName string) {
	if r := recover(); r != nil {
		fmt.Printf("%s : PANIC Defered : %v\n", functionName, r)

		// Capture the stack trace
		buf := make([]byte, 10000)
		runtime.Stack(buf, false)

		fmt.Printf("%s : Stack Trace : %s", functionName, string(buf))

		if err != nil {
			*err = fmt.Errorf("%v", r)
		}
	}
}

func topicName(code string) string {
	switch topic := code; topic {
	case "latest":
		return "Latest"
	case "business":
		return "Business"
	case "politics":
		return "Politics"
	case "entertainment":
		return "Entertainment"
	case "tech":
		return "Tech"
	case "sport":
		return "Sport"
	case "gossips":
		return "Gossips"
	case "art_culture":
		return "Art & Culture"
	case "film":
		return "Film"
	case "food":
		return "Food"
	case "music":
		return "Music"
	case "science":
		return "Science"
	case "photography":
		return "Photography"
	case "travel":
		return "Travel"
	case "style":
		return "Style"
	case "Health":
		return "Health"
	case "Media":
		return "Media"
	case "lgbt":
		return "LGBT+"
	default:
		return "All"
	}
}

func send(email string) (err error) {
	session, databaseName, err := db.GetSession()

	if err != nil {
		panic(err)
	}

	defer session.Close()

	usersCollection := session.DB(databaseName).C("readers")
	headlinesCollection := session.DB(databaseName).C("headlines")
	channelsCollection := session.DB(databaseName).C("channels")

	users := []User{}
	headlines := []News{}
	channels := []Channel{}
	mapChannels := map[string]string{}

	err = channelsCollection.Find(bson.M{}).All(&channels)
	if err != nil {
		panic(err)
	}

	for _, channel := range channels {
		mapChannels[channel.Code] = channel.Name
	}

	queryUsers := bson.M{
		"email":   email,
		"next_at": bson.M{"$lte": time.Now().UTC()},
	}

	err = usersCollection.Find(queryUsers).All(&users)
	if err != nil {
		panic(err)
	}

	for _, user := range users {
		log.Println("Name:", user.Email)

		// TODO: Get headlines
		start, _ := dateparse.ParseLocal(time.Now().UTC().String())

		last3hours := start.Add(time.Hour * time.Duration(-12)).UTC()
		lastDeliveredAt := user.DeliveredAt.UTC()
		if lastDeliveredAt.IsZero() {
			lastDeliveredAt = last3hours
		}

		queryHeadlines := bson.M{
			"channel":      bson.M{"$in": user.Channels},
			"section":      bson.M{"$in": user.Topics},
			"created_at":   bson.M{"$gte": lastDeliveredAt},
			"position_idx": bson.M{"$lte": 5},
			"history_idx":  bson.M{"$elemMatch": bson.M{"$gte": 0, "$lte": 5}},
		}

		err := headlinesCollection.Find(queryHeadlines).Sort("-created_at").Limit(100).All(&headlines) // .Sort("-created_at", "section", "position_idx")
		if err != nil {
			panic(err)
		}

		limiter := make(map[string]int, 0)
		tmp := make([]map[string]interface{}, 0)
		titles := make([]string, 0)
		insideChannels := make([]string, 0)

		type Items struct {
			Topic string
			Items []News
		}
		Headlines := map[string][]Items{}

		for _, item := range headlines {
			alreadyExists := false
			items := make([]News, 0)

			if limiter[item.Channel] <= 5 {
				for _, title := range titles {
					weightTitle := jaro.Jaro(title, item.Title)
					if weightTitle >= 0.9 {
						alreadyExists = true
						break
					}
				}

				titles = append(titles, item.Title)

				if !util.Contains(insideChannels, mapChannels[item.Channel]) {
					insideChannels = append(insideChannels, mapChannels[item.Channel])
				}

				if !alreadyExists {
					u, err := url.Parse(item.Link)
					if err != nil {
						item.Hostname = item.Link
					}
					item.Hostname = u.Hostname()
					items := append(items, item)
					section := topicName(item.Section)
					Headlines[section] = append(Headlines[section], Items{
						section,
						items,
					})
					tmp = append(tmp, structs.Map(item))
					limiter[item.Channel]++
				}
			}
		}

		fmt.Println("Number of news", len(tmp))

		if len(tmp) > 0 {

			unsubscribeToken := util.SignToken(structs.Map(user), "unsubscribe")

			data := struct {
				Token     string
				Website   string
				UserID    string
				Email     string
				Headlines map[string][]Items
			}{
				Token:     unsubscribeToken,
				Website:   util.GetEnv("WEBSITE_URL", ""),
				UserID:    user.ID.Hex(),
				Email:     user.Email,
				Headlines: Headlines,
			}

			// fmt.Println("tmp", tmp)

			subject := "Latest news from " + strings.Join(insideChannels[:len(insideChannels)-2], ", ") + " and " + insideChannels[len(insideChannels)-1]
			reStr := regexp.MustCompile("/,([^,]*)$/")
			subject = reStr.ReplaceAllString(subject, " and $1")
			fmt.Println("subject", subject)

			resp, err := ses.SendEmailUsingTemplate("newsletter_001.html", user.Email, subject, data)
			if err != nil {
				panic(err)
			}

			fmt.Println("Envelope response", resp)
		}

		converdted := make(map[string]interface{})
		converdted["days"] = user.Days
		converdted["hours"] = user.Hours
		converdted["timezone"] = user.Timezone
		converdted["is_unsubscribed"] = user.UnsubscribedAt.IsZero()

		nextAt, err := util.CalculateNextAt(converdted)

		userChange := mgo.Change{
			Update: bson.M{
				"$set": bson.M{
					"delivered_at": time.Now(),
					"next_at":      nextAt,
				},
			},
			ReturnNew: false,
			Upsert:    false,
		}

		userDoc := bson.M{}
		_, err = usersCollection.Find(bson.M{
			"email": user.Email,
		}).Apply(userChange, &userDoc)
		if err != nil {
			if debug {
				log.Printf("RunQuery : ERROR : %s\n", err)
			}
		}

		// TODO: Save last delivered_at
		// TODO: Calculate next_at
	}

	return err
}

// Execute main function
func Execute() {
	// Log memory usage every n seconds
	if globalOptions.LogMode {
		debug = true
		database.Debug = debug
	}

	if globalOptions.MemoryMode {
		go logAllocMemory()
	}

	if debug {
		log.Println("Process init.")
	}

	if globalOptions.Email != "" {
		err := db.CreateConnection()

		if err != nil {
			panic(err)
		}

		defer db.CloseSession()
		send(globalOptions.Email)
	} else {
		fmt.Println("Tip: Use -help to display available options.")
	}
}
