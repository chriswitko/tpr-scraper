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
var dbNAME string
var debug bool

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

// Newspaper is a collection of news
type Newspaper []News

// FeedChannel is a collection of channels
type FeedChannel struct {
	Code        string    `bson:"code"`
	ProcessedAt time.Time `bson:"processed_at"`
}

// FeedSection is a part of feed
type FeedSection struct {
	Code      string
	Category  string
	Channel   string
	Format    string
	Source    string
	RawSource string `bson:"raw_source"`
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
	if len(n.all()) > 0 {
		fmt.Println(n)
	}
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

func display(newspaper *Newspaper) {
	newspaper.print()
}

func publish(newspaper *Newspaper) {
	if debug {
		log.Println("Publishing using bulk method")
	}
	session, err := mgo.Dial(dbURI)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	i := 0

	collection := session.DB(dbNAME).C("items_bulk")
	bulk := collection.Bulk()

	all := newspaper.all()
	prevChannel := ""
	for _, news := range all {
		i++
		bulk.Upsert(bson.M{"hash": news.Hash}, news)
		if prevChannel != news.Channel {
			updateChannel(FeedChannel{
				Code: news.Channel,
			}, session)
			prevChannel = news.Channel
		}
	}
	bulk.Unordered()
	_, err = bulk.Run()
	if err != nil {
		panic(err)
	}

	if debug {
		log.Println("All queries completed")
	}
}

func publishSynch(newspaper *Newspaper) {
	if debug {
		log.Println("Publishing using queue method")
	}
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
	if debug {
		log.Println("All queries completed")
	}
}

func loadEnv() {
	e := godotenv.Load()
	if e != nil {
		panic("Error loading .env file")
	}
}

func loadDBURI() (string, string) {
	dbURI := os.Getenv("DB_URI")
	dbNAME := os.Getenv("DB_NAME")
	if dbURI == "" || dbNAME == "" {
		panic("Missing DB setting in .env file")
	}
	return dbURI, dbNAME
}

func getAllChannels(newspaper *Newspaper) (err error) {
	if debug {
		log.Println("DB_URI", dbURI)
		log.Println("DB_NAME", dbNAME)
	}

	session, err := mgo.Dial(dbURI)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	collection := session.DB(dbNAME).C("channels")
	result := Feed{}

	localTime := time.Now()
	dur, _ := time.ParseDuration("5m")

	err = collection.Find(bson.M{
		"lab": true,
		"processed_at": bson.M{
			"$lte": localTime.Add(-dur),
		},
	}).All(&result)
	if err != nil {
		panic(err)
	}

	for _, elem := range result {
		for _, section := range elem.Sections {
			if debug {
				log.Println("Name:", elem.Name)
				log.Println("Code:", elem.Code)
				log.Println("Section:", section.Code)
			}
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
	if debug {
		log.Println("Section URL:", section.RawSource)
	}
	c := colly.NewCollector()

	news := &News{}

	var sectionNews []News

	// On every a element which has href attribute call callback
	c.OnHTML(section.Pattern, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
		normStr1, _, _ := transform.String(t, e.Text)
		title := strings.TrimSpace(strings.Trim(normStr1, "\u00a0"))
		f := func(c rune) bool {
			return !unicode.IsLetter(c) && !unicode.IsNumber(c)
		}
		title = strings.TrimFunc(title, f)
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
			sectionNews = append(sectionNews, *news)
			*newspaper = append(*newspaper, *news)
		}
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		if debug {
			log.Println("Visiting", r.URL.String())
		}
	})

	c.Visit(section.RawSource)
	if debug {
		log.Printf("Total number of news: %d\n", len(sectionNews))
	}
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

func updateChannel(feedChannel FeedChannel, mongoSession *mgo.Session) {
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()

	// Get a collection to execute the query against.
	channels := sessionCopy.DB(dbNAME).C("channels")

	localTime := time.Now()
	utcTime := localTime.UTC() //.Format(time.RFC3339)
	feedChannel.ProcessedAt = utcTime

	channelChange := mgo.Change{
		Update: bson.M{
			"$set": feedChannel,
		},
		ReturnNew: false,
		Upsert:    false,
	}

	channelDoc := bson.M{}
	_, channelErr := channels.Find(bson.M{"code": feedChannel.Code}).Apply(channelChange, &channelDoc)
	if channelErr != nil {
		if debug {
			log.Printf("RunQuery : ERROR : %s\n", channelErr)
		}
		return
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
	items := sessionCopy.DB(dbNAME).C("items_beta")

	if debug {
		log.Printf("RunQuery : %d : Executing\n", query)
	}

	// Retrieve the list of stations.
	itemChange := mgo.Change{
		Update:    news,
		ReturnNew: false,
		Upsert:    true,
	}

	doc := bson.M{}
	_, err := items.Find(bson.M{"hash": news.Hash}).Apply(itemChange, &doc)
	if err != nil {
		if debug {
			log.Printf("RunQuery : ERROR : %s\n", err)
		}
		return
	}

	if debug {
		log.Printf("RunQuery : SUCCESS: %d \n", query)
	}
}

func main() {
	loadEnv()
	dbURI, dbNAME = loadDBURI()

	// counter := counters.AlertCounter(10)

	var logMode bool
	var testMode bool
	var allMode bool
	var saveMode bool
	var displayMode bool
	var memoryMode bool
	var url string
	var pattern string
	var newspaper Newspaper

	flag.BoolVar(&logMode, "log", false, "Turn on/off logs")
	flag.BoolVar(&testMode, "test", false, "Test mode to verify pattern, -u and -p attributes are required")
	flag.BoolVar(&allMode, "all", false, "Process all channels")
	flag.BoolVar(&saveMode, "save", false, "Save all results to database")
	flag.BoolVar(&displayMode, "display", false, "Display all results on the screen")
	flag.BoolVar(&memoryMode, "memory", false, "Display memory benchmarks")
	flag.StringVar(&url, "u", "", "The URL to fetch")
	flag.StringVar(&pattern, "p", "", "Pattern to search on website")
	flag.Parse()

	// Log memory usage every n seconds
	if logMode {
		debug = true
	}

	if memoryMode {
		go logAllocMemory()
	}

	if debug {
		log.Println("Process init.")
	}

	if testMode {
		fmt.Println("This is a test mode")
		if url == "" || pattern == "" {
			fmt.Println("URL and Pattern are required")
			os.Exit(1)
		}
		section := FeedSection{
			RawSource: url,
			Pattern:   pattern,
		}
		processSection(section, &newspaper)
	} else if allMode {
		getAllChannels(&newspaper)
	} else {
		fmt.Println("Tip: Use -help to display available options.")
	}

	if saveMode {
		publish(&newspaper)
	}

	if displayMode {
		display(&newspaper)
	}

	if debug {
		log.Println("Process done.")
	}
}
