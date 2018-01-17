package collect

import (
	"amp"
	"archive/zip"
	"cdn"
	"crypto/md5"
	"database"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/asciimoo/colly"
	"github.com/disintegration/imaging"
	"github.com/imdario/mergo"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	version string = "master"
)

var debug bool

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
}

// Newspaper is a collection of news
type Newspaper []News

// FeedChannel is a collection of channels
type FeedChannel struct {
	Code            string    `bson:"code"`
	ProcessedAt     time.Time `bson:"processed_at"`
	LastImportTotal int       `bson:"last_import_total"`
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

// MongoConnection for all endpoints
var mu = &sync.Mutex{}

// Options - a global settings
type Options struct {
	LogMode     bool
	TestMode    bool
	AllMode     bool
	SaveMode    bool
	DisplayMode bool
	MemoryMode  bool
	UploadMode  bool
	Clusters    int
	Limit       int
	URL         string
	Pattern     string
	Channels    string
	Sections    string
}

var globalOptions Options
var db database.MongoConnection
var newspaper Newspaper

// var wg sync.WaitGroup

// SetOptions manage global options
func SetOptions(options Options) {
	mergo.Merge(&globalOptions, options)
}

func (n *Newspaper) print() error {
	if len(n.all()) > 0 {
		fmt.Println(n)
	}
	return nil
}

func (n *Newspaper) all() []News {
	return *n
}

func display(newspaper *Newspaper) {
	newspaper.print()
}

func publish(newspaper *Newspaper) {
	if debug {
		log.Println("Publishing using bulk method")
	}
	session, databaseName, err := db.GetSession()

	if err != nil {
		panic(err)
	}

	defer session.Close()

	i := 0

	collection := session.DB(databaseName).C("items_bulk")
	bulk := collection.Bulk()

	all := newspaper.all()
	prevChannel := ""
	for _, news := range all {
		i++
		bulk.Upsert(bson.M{"hash": news.Hash}, bson.M{"$set": news, "$addToSet": bson.M{"history": news.Position}})
		if prevChannel != news.Channel {
			updateChannel(FeedChannel{
				Code: news.Channel,
			}, session, &databaseName)
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
	session, databaseName, err := db.GetSession()

	if err != nil {
		panic(err)
	}

	defer session.Close()

	i := 0

	all := newspaper.all()
	prevChannel := ""
	// if limit >= 0 {
	// 	all = all[:limit]
	// }
	waitGroup.Add(len(all))

	output := make(map[string]int)

	for _, m := range all {
		output[m.Channel] = output[m.Channel] + 1
	}

	for _, news := range all {
		i++
		// TODO: Replace with bulk updates
		go updateItemSafe(i, news, &waitGroup, session, &databaseName)

		// Updating channel
		if prevChannel != news.Channel {
			updateChannel(FeedChannel{
				Code:            news.Channel,
				LastImportTotal: output[news.Channel],
			}, session, &databaseName)
			prevChannel = news.Channel
			i = 0
		}
	}
	waitGroup.Wait()
	if debug {
		log.Println("All queries completed")
	}
}

func getAllChannels(newspaper *Newspaper, channels string, sections string, limit int) (err error) {
	session, databaseName, err := db.GetSession()

	if err != nil {
		panic(err)
	}

	defer db.CloseSession() // TODO: TEST THIS DEFER!!!
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.

	collection := session.DB(databaseName).C("channels")
	result := Feed{}

	localTime := time.Now()
	dur, _ := time.ParseDuration("5m")

	query := bson.M{}

	f := func(c rune) bool {
		return unicode.IsSpace(c)
	}

	if len(channels) > 0 || len(sections) > 0 {
		if len(channels) > 0 {
			prepChannels := strings.Split(channels, ",")
			if len(prepChannels) > 0 {
				for i := 0; i < len(prepChannels); i++ {
					prepChannels[i] = strings.TrimFunc(prepChannels[i], f)
				}
				query["code"] = bson.M{
					"$in": prepChannels,
				}
				fmt.Println("channels", prepChannels)
			}
		}

		if len(sections) > 0 {
			prepSections := strings.Split(sections, ",")
			if len(prepSections) > 0 {
				for i := 0; i < len(prepSections); i++ {
					prepSections[i] = strings.TrimFunc(prepSections[i], f)
				}
				query["sections.code"] = bson.M{
					"$in": prepSections,
				}
			}
		}
	} else {
		query = bson.M{
			"lab": true,
			"$and": []bson.M{
				bson.M{"$or": []bson.M{
					bson.M{"processed_at": bson.M{"$exists": false}},
					bson.M{"processed_at": bson.M{"$lte": localTime.Add(-dur)}},
				},
				},
			},
		}
	}

	err = collection.Find(query).All(&result)
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
				out1 <- processSection(section, newspaper, limit)
			}()
			<-out1
		}
	}

	return err
}

func standardizeSpaces(s string) string {
	// TODO: Detect canonical and amp urls
	// TODO: Detect primary image url and sizes
	return strings.Join(strings.Fields(s), " ")
}

// String minifier, remove whitespaces
func processSection(section FeedSection, newspaper *Newspaper, limit int) (result Newspaper) {
	if debug {
		log.Println("Section URL:", section.RawSource)
	}

	if section.RawSource == "" {
		return *newspaper
	}

	var sectionNews []News
	position := 1

	if section.Format == "html" {
		c := colly.NewCollector()
		c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5})

		// news := &News{}

		// On every a element which has href attribute call callback
		c.OnHTML(section.Pattern, func(e *colly.HTMLElement) {
			if position > limit {
				return
			}
			link := e.Attr("href")

			title := standardizeSpaces(e.Text)
			f := func(c rune) bool {
				return unicode.IsSpace(c)
			}
			title = strings.TrimFunc(title, f)
			title = strings.TrimSpace(title)
			if len(strings.Trim(title, "")) > 0 && len(strings.Trim(link, "")) > 0 {
				fmt.Println(title)
				fmt.Println(" - ", link)

				localTime := time.Now()
				utcTime := localTime.UTC() //.Format(time.RFC3339)

				hasher := md5.New()
				hasher.Write([]byte(link))

				news := News{
					Hash:        hex.EncodeToString(hasher.Sum(nil)),
					Title:       title,
					Description: "",
					Link:        link,
					Section:     section.Category,
					Channel:     section.Channel,
					CreatedAt:   utcTime,
					Position:    position,
				}

				if news.Title != "" {
					sectionNews = append(sectionNews, news)
					*newspaper = append(*newspaper, news)
					position++
				}
			}
		})

		// Before making a request print "Visiting ..."
		c.OnRequest(func(r *colly.Request) {
			if debug {
				log.Println("Visiting", r.URL.String())
			}
		})

		c.OnError(func(r *colly.Response, err error) {
			fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
		})

		c.Visit(section.RawSource)

		c.Wait()
	} else if section.Format == "rss" {
		fp := gofeed.NewParser()
		feed, err := fp.ParseURL(section.RawSource)

		if err != nil {
			return *newspaper
		}

		if feed == nil {
			return *newspaper
		}

		for _, item := range feed.Items[:globalOptions.Limit] {
			fmt.Println(item.Title)
			fmt.Println(" - ", item.Link)

			localTime := time.Now()
			utcTime := localTime.UTC() //.Format(time.RFC3339)

			hasher := md5.New()
			hasher.Write([]byte(item.Link))

			news := News{
				Hash:        hex.EncodeToString(hasher.Sum(nil)),
				Title:       strings.TrimSpace(item.Title),
				Description: "",
				// OriginalImageURL: item.Image.URL,
				Link:      item.Link,
				Section:   section.Category,
				Channel:   section.Channel,
				CreatedAt: utcTime,
				Position:  position,
			}

			if news.Title != "" {
				sectionNews = append(sectionNews, news)
				*newspaper = append(*newspaper, news)
				position++
			}

			if len(sectionNews) >= 10 {
				break
			}
		}
	}

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

func updateChannel(feedChannel FeedChannel, mongoSession *mgo.Session, databaseName *string) {
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy := mongoSession.Clone()
	defer sessionCopy.Close()

	// Get a collection to execute the query against.
	channels := sessionCopy.DB(*databaseName).C("channels")

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

func updateItem(query int, news News, waitGroup *sync.WaitGroup, mongoSession *mgo.Session, databaseName *string) {
	mu.Lock()
	defer mu.Unlock()
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	defer waitGroup.Done()

	// Request a socket connection from the session to process our query.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy := mongoSession.Clone()
	defer sessionCopy.Close()

	// Get a collection to execute the query against.
	items := sessionCopy.DB(*databaseName).C("headlines")

	if debug {
		log.Printf("RunQuery : %d : Executing\n", query)
	}

	// Retrieve the list of stations.
	itemChange := mgo.Change{
		Update:    bson.M{"$set": news}, // chris change
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

// source: https://godoc.org/github.com/tensorflow/tensorflow/tensorflow/go#example-package
func filesExist(files ...string) error {
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("unable to stat %s: %v", f, err)
		}
	}
	return nil
}

func download(URL, filename string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func unzip(dir, zipfile string) error {
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		src, err := f.Open()
		if err != nil {
			return err
		}
		log.Println("Extracting", f.Name)
		dst, err := os.OpenFile(filepath.Join(dir, f.Name), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		dst.Close()
	}
	return nil
}

func appendIfMissing(slice []int, i int) []int {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func uniqueFileName(filename string) string {
	out := uuid.NewV4()
	extension := filepath.Ext(filename)

	if strings.Contains(extension, ".jpg") {
		extension = ".jpg"
	} else if strings.Contains(extension, ".png") {
		extension = ".png"
	} else if strings.Contains(extension, ".gif") {
		extension = ".gif"
	} else if strings.Contains(extension, ".webp") {
		extension = ".webp"
	} else {
		extension = ".jpg"
	}
	return out.String() + extension
}

func updateItemSafe(query int, news News, waitGroup *sync.WaitGroup, mongoSession *mgo.Session, databaseName *string) {
	mu.Lock()
	defer mu.Unlock()
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	defer waitGroup.Done()

	// Request a socket connection from the session to process our query.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy := mongoSession.Clone()
	defer sessionCopy.Close()

	// Get a collection to execute the query against.
	items := sessionCopy.DB(*databaseName).C("headlines")

	if debug {
		log.Printf("RunQuery : %d : Executing\n", query)
	}

	// Retrieve the list of stations.

	doc := News{}
	err := items.Find(bson.M{"hash": news.Hash}).One(&news)

	news.History = appendIfMissing(news.History, query)

	if err != nil {
		if debug {
			log.Printf("RunQuery : ERROR : %s\n", err)
		}
		fmt.Println("Item does not exists")

		links, _ := amp.Parse(news.Link)

		if links != nil && links.Canonical != "" {
			news.CanonicalURL = links.Canonical
		}

		if links != nil && links.AMP != "" {
			news.AmpURL = links.AMP
		}

		if links != nil && links.Image != "" {
			news.OriginalImageURL = links.Image
			if globalOptions.UploadMode {

				temppath := "./tmp"
				filename := uniqueFileName(links.Image)
				filepath := temppath + "/" + filename
				download(links.Image, filepath)
				news.ImageUUID = filename

				src, err := imaging.Open(filepath)
				if err != nil {
					log.Fatalf("Open failed: %v", err)
				}

				b := src.Bounds()
				news.ImageWidth = b.Max.X
				news.ImageHeight = b.Max.Y

				// Crop the original image to 350x350px size using the center anchor.
				inpImageSmall := imaging.Resize(src, 0, 111, imaging.Lanczos)
				dstImageSmall := imaging.CropAnchor(inpImageSmall, 111, 74, imaging.Center)
				err = imaging.Save(dstImageSmall, temppath+"/s_"+filename)
				if err != nil {
					log.Fatalf("Save failed: %v", err)
				}
				inpImageSmallSquare := imaging.Resize(src, 0, 158, imaging.Lanczos)
				dstImageSmallSquare := imaging.CropAnchor(inpImageSmallSquare, 158, 158, imaging.Center)
				err = imaging.Save(dstImageSmallSquare, temppath+"/ssq_"+filename)
				if err != nil {
					log.Fatalf("Save failed: %v", err)
				}
				inpImageMedium := imaging.Resize(src, 506, 0, imaging.Lanczos)
				err = imaging.Save(inpImageMedium, temppath+"/m_"+filename)
				if err != nil {
					log.Fatalf("Save failed: %v", err)
				}
				inpImageMediumSquare := imaging.Resize(src, 0, 506, imaging.Lanczos)
				dstImageMediumSquare := imaging.CropAnchor(inpImageMediumSquare, 506, 506, imaging.Center)
				err = imaging.Save(dstImageMediumSquare, temppath+"/msq_"+filename)
				if err != nil {
					log.Fatalf("Save failed: %v", err)
				}
				inpImageLarge := imaging.Resize(src, 800, 0, imaging.Lanczos)
				err = imaging.Save(inpImageLarge, temppath+"/l_"+filename)
				if err != nil {
					log.Fatalf("Save failed: %v", err)
				}
			}
		}
	} else {
		fmt.Println("Item exists")
	}

	itemChange := mgo.Change{
		Update:    bson.M{"$set": news},
		ReturnNew: false,
		Upsert:    true,
	}

	// TODO: Replace this part with Bulk
	_, err = items.Find(bson.M{"hash": news.Hash}).Apply(itemChange, &doc)
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

// Execute main function
func Execute() {

	dir := "./"

	var (
		temp = filepath.Join(dir, "tmp")
	)

	if filesExist(temp) != nil {
		log.Println("Did not find temp folder in ", dir)
		if err := os.MkdirAll(temp, 0755); err != nil {
			panic("Could not create a temp folder")
		}
	}

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

	if globalOptions.TestMode {
		if globalOptions.URL == "" || globalOptions.Pattern == "" {
			fmt.Println("Missing flags. --url and --pattern are required.")
			os.Exit(0)
		}
		section := FeedSection{
			RawSource: globalOptions.URL,
			Pattern:   globalOptions.Pattern,
		}
		processSection(section, &newspaper, globalOptions.Limit)
	} else if globalOptions.AllMode {
		err := db.CreateConnection()

		if err != nil {
			panic(err)
		}

		// defer db.CloseSession()
		getAllChannels(&newspaper, globalOptions.Channels, globalOptions.Sections, globalOptions.Limit)
	} else {
		fmt.Println("Tip: Use -help to display available options.")
	}

	if globalOptions.SaveMode {
		// publish(&newspaper)
		publishSynch(&newspaper)
	}

	if globalOptions.DisplayMode {
		display(&newspaper)
	}

	if globalOptions.UploadMode {
		cdn.Upload("thepressreview", "images/", globalOptions.Clusters, "us-east-1", "public-read", "./tmp/", "./uploaded/")
	}

	if debug {
		log.Println("Process done.")
	}
}
