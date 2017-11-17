[] Add multiple patterns per site .area a,.area2 a
[] Add -channels=gazetapl,bbc and -sections=latest
[] Add -exclude
[] Test parsing XML file (RSS, flag: -rss, go run main.go -test -rss -u "http://rss.cnn.com/rss/edition.rss" -p="item" -display -log)
[] Write simple tests (https://github.com/eaigner/shield/blob/master/en_tokenizer_test.go)
[] Move DB init connection to main function or struct with custom funcs DB.Connect, DB.BulkNews etc.
[] Replace var newspaper Newspaper this with newspaper := make([]*News, 0)

// Example: go run main.go -test -u http://www.gazeta.pl/0,0.html -p=".mt_list a"
// go run main.go -test -u https://nytimes.com -p=".story a"
// go run main.go -test -u "http://edition.cnn.com" -p "h3.cd__headline a" -display -log
// go run main.go -test -u "http://edition.cnn.com/entertainment" -p=".cd__headline a" -display -log

https://github.com/asciimoo/colly/blob/master/_examples/hackernews_comments/hackernews_comments.go
https://godoc.org/gopkg.in/mgo.v2#Collection.Find
https://github.com/ungerik/go-dry/blob/master/string.go
https://gobyexample.com