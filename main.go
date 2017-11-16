package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	counters "./utils"

	"github.com/asciimoo/colly"
	"github.com/joho/godotenv"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	version string = "master"
)

var dbURI string

// News is part of feeditem
type News struct {
	Title       string    `bson:"title"`
	Description string    `bson:"description"`
	Link        string    `bson:"url"`
	Channel     string    `bson:"channel"`
	Section     string    `bson:"section"`
	CreatedAt   time.Time `bson:"created_at"`
	Hash        string    `bson:"hash"`
}

// Newspaper is collection of news
type Newspaper []News

// FeedSection is part of feed
type FeedSection struct {
	Code      string
	Category  string
	Channel   string
	Format    string
	Source    string
	Rawsource string `bson:"rawsource"`
	Pattern   string
}

// FeedItem Single line
type FeedItem struct {
	Name     string
	Code     string
	Pattern  string
	Link     string
	Sections []FeedSection
}

// Feed is a collection for channels
type Feed []FeedItem

// var wg sync.WaitGroup

func (n *Newspaper) print() error {
	fmt.Println(n)
	return nil
}

func (n *Newspaper) all() []News {
	return *n
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func filterNewLines(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 0x000A, 0x000B, 0x000C, 0x000D, 0x0085, 0x2028, 0x2029:
			return -1
		default:
			return r
		}
	}, s)
}

func stripSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// if the character is a space, drop it
			return -1
		}
		// else keep it in the string
		return r
	}, str)
}

func finalPrint(newspaper *Newspaper) {
	newspaper.print()

	session, err := mgo.Dial(dbURI)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	i := 0

	collection := session.DB("thepressreview-prod").C("items_bulk")
	bulk := collection.Bulk()

	all := newspaper.all()
	for _, news := range all {
		i++
		bulk.Upsert(bson.M{"hash": news.Hash}, news)
	}
	bulk.Unordered()
	_, err = bulk.Run()
	if err != nil {
		panic(err)
	}
	log.Println("All Queries Completed")
}

func finalPrintSynch(newspaper *Newspaper) {
	newspaper.print()

	var waitGroup sync.WaitGroup
	session, err := mgo.Dial(dbURI)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	i := 0

	all := newspaper.all()
	waitGroup.Add(len(all))
	for _, news := range all {
		i++
		go runQuery(i, news, &waitGroup, session)
	}
	waitGroup.Wait()
	log.Println("All Queries Completed")
}

func loadEnv() {
	e := godotenv.Load()
	if e != nil {
		log.Fatal("Error loading .env file")
	}
}

func loadDBURI() string {
	dbURI := os.Getenv("DB_URI")
	if dbURI == "" {
		log.Fatal("Missing DB setting in .env file")
	}
	return dbURI
}

func getAllChannels(newspaper *Newspaper) (err error) {
	fmt.Println("uri", dbURI)
	// TODO: Move DB init connection to main function or struct with custom funcs DB.Connect, DB.BulkNews etc.
	session, err := mgo.Dial(dbURI)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	collection := session.DB("thepressreview-prod").C("channels")
	result := Feed{}

	err = collection.Find(bson.M{"lab": true}).All(&result)
	if err != nil {
		log.Fatal(err)
	}

	for _, elem := range result {
		for _, section := range elem.Sections {
			fmt.Println("Name:", elem.Name)
			fmt.Println("Section:", section.Code)
			section.Channel = elem.Code
			out1 := make(chan Newspaper)
			go func() {
				out1 <- processSection(section, newspaper)
			}()
			<-out1
		}
	}

	return err
}

func processSection(section FeedSection, newspaper *Newspaper) (result Newspaper) {
	fmt.Println("Section URL:", section.Rawsource)
	c := colly.NewCollector()

	news := &News{}

	// On every a element which has href attribute call callback
	c.OnHTML(section.Pattern, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
		normStr1, _, _ := transform.String(t, e.Text)
		title := strings.TrimSpace(strings.Trim(normStr1, "\u00a0"))
		if len(title) > 0 {
			localTime := time.Now()
			utcTime := localTime.UTC() //.Format(time.RFC3339)

			hasher := md5.New()
			hasher.Write([]byte(link))

			*news = News{
				Hash:        hex.EncodeToString(hasher.Sum(nil)),
				Title:       title,
				Description: "",
				Link:        link,
				Section:     section.Category,
				Channel:     section.Channel,
				CreatedAt:   utcTime,
			}
			*newspaper = append(*newspaper, *news)
		}
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	c.Visit(section.Rawsource)
	return *newspaper
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

func runQuery(query int, news News, waitGroup *sync.WaitGroup, mongoSession *mgo.Session) {
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	defer waitGroup.Done()

	// Request a socket connection from the session to process our query.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()

	// Get a collection to execute the query against.
	collection := sessionCopy.DB("thepressreview-prod").C("items_beta")

	log.Printf("RunQuery : %d : Executing\n", query)

	// Retrieve the list of stations.
	change := mgo.Change{
		Update:    news,
		ReturnNew: false,
		Upsert:    true,
	}

	doc := bson.M{}
	_, err := collection.Find(bson.M{"hash": news.Hash}).Apply(change, &doc)
	// err := collection.Find(nil).All(&buoyStations)
	if err != nil {
		log.Printf("RunQuery : ERROR : %s\n", err)
		return
	}

	log.Printf("RunQuery : SUCCESS: %d \n", query)
}

func main() {
	loadEnv()
	dbURI = loadDBURI()

	counter := counters.AlertCounter(10)

	fmt.Printf("Counter: %d\n", counter)

	var logMode bool
	var testMode bool
	var url string
	var pattern string
	// TODO: Replace this with newspaper := make([]*News, 0)
	var newspaper Newspaper

	flag.BoolVar(&logMode, "log", false, "log mode on/off")
	flag.BoolVar(&testMode, "test", false, "test mode on/off")
	flag.StringVar(&url, "u", "", "website url")
	flag.StringVar(&pattern, "p", "", "pattern for website")
	flag.Parse()

	// Log memory usage every n seconds
	if logMode {
		go logAllocMemory()
	}

	localTime := time.Now()
	utcTime := localTime.UTC().Format(time.RFC3339)
	fmt.Println("Current time UTC", utcTime)

	if testMode {
		fmt.Println("This is a test mode")
		if url == "" || pattern == "" {
			log.Println("URL and Pattern are required")
			os.Exit(1)
		}
		section := FeedSection{
			Rawsource: url,
			Pattern:   pattern,
		}
		processSection(section, &newspaper)
	} else {
		getAllChannels(&newspaper)
	}

	// TODO: Add option -save to save to database or just display on screen
	finalPrint(&newspaper)
}
