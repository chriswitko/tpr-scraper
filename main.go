package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/asciimoo/colly"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	version string = "master"
)

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

// News is part of feeditem
type News struct {
	Title string
	Link  string
}

// Newspaper is collection of news
type Newspaper []News

// FeedSection is part of feed
type FeedSection struct {
	Code      string
	Category  string
	Format    string
	Source    string
	Rawsource string
	Pattern   string
}

// FeedItem Single line
type FeedItem struct {
	Name     string
	Pattern  string
	Link     string
	Sections []FeedSection
}

// Feed is a collection for channels
type Feed []FeedItem

// prints arg1, arg2
func f2(arg1 Newspaper) {
	// fmt.Println(arg1)
}

func finalPrint(newspaper *Newspaper) {
	fmt.Println(*newspaper)
}

func main() {
	// Log memory usage every n seconds
	go func() {
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC = %v\n\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
			time.Sleep(1 * time.Second)
		}
	}()

	var testMode bool
	var url string
	var pattern string
	// flag.StringVar(&itemID, "id", "", "hackernews post id")
	flag.BoolVar(&testMode, "test", false, "test mode on/off")
	flag.StringVar(&url, "u", "", "website url")
	flag.StringVar(&pattern, "p", "", "pattern for website")
	flag.Parse()

	fmt.Println("testMode", testMode)

	var newspaper Newspaper

	// Example: go run main.go -test -u http://www.gazeta.pl/0,0.html -p=".mt_list a"
	// go run main.go -test -u https://nytimes.com -p=".story a"
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
		getLinks(section, &newspaper)
	} else {
		getAllLinks(&newspaper)
	}

	finalPrint(&newspaper)
}

func getAllLinks(newspaper *Newspaper) {
	session, err := mgo.Dial("mongodb://suburb:db$studio$2017@159.203.95.97:27017/thepressreview-prod?authMechanism=SCRAM-SHA-1&authSource=admin")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	db := session.DB("thepressreview-prod").C("channels")
	// Instantiate default collector
	result := Feed{}
	err = db.Find(bson.M{"lab": true}).All(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Name:", result)

	for _, elem := range result {
		for _, section := range elem.Sections {
			fmt.Println("Name:", elem.Name)
			fmt.Println("Section:", section.Code)
			out1 := make(chan Newspaper)
			go func() {
				out1 <- getLinks(section, newspaper)
			}()
			f2(<-out1)
		}
	}
}

func getLinks(section FeedSection, newspaper *Newspaper) (result Newspaper) {
	fmt.Println("Section URL:", section.Rawsource)
	c := colly.NewCollector()
	// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
	// c.AllowedDomains = []string{"gazeta.pl"}

	// On every a element which has href attribute call callback
	c.OnHTML(section.Pattern, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
		normStr1, _, _ := transform.String(t, e.Text)
		title := strings.TrimSpace(strings.Trim(normStr1, "\u00a0"))
		if len(title) > 0 {
			// fmt.Printf("Link found: %q -> %s\n", title, link)
			news := News{
				Title: title,
				Link:  link,
			}
			*newspaper = append(*newspaper, news)
		}
		// Visit link found on page
		// Only those links are visited which are in AllowedDomains
		// c.Visit(e.Request.AbsoluteURL(link))
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping on https://hackerspaces.org
	// c.Visit("http://www.gazeta.pl/0,0.html")
	c.Visit(section.Rawsource)
	return *newspaper
	// ch <- fmt.Sprintf("done")
	// { name: 'headlines', template: '#text_topnews a', url: 'https://www.wp.pl' },
}
