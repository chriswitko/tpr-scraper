[] Test layer -test - to test url + pattern

// Example: go run main.go -test -u http://www.gazeta.pl/0,0.html -p=".mt_list a"
// go run main.go -test -u https://nytimes.com -p=".story a"

https://github.com/asciimoo/colly/blob/master/_examples/hackernews_comments/hackernews_comments.go
https://godoc.org/gopkg.in/mgo.v2#Collection.Find
https://github.com/ungerik/go-dry/blob/master/string.go
https://gobyexample.com

// Add multiple patterns per site
// Add -exclude
// use bulk := coll.Bulk() -> https://github.com/go-mgo/mgo/pull/336/files?diff=split
// -> https://stackoverflow.com/questions/23583198/partial-update-using-mgo

// TODO: Save last_processed_at per channel