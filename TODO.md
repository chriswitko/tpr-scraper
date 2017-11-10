[] Test layer -test - to test url + pattern

  Item.findOneAndUpdate({
    channel: args.channel,
    section: args.section,
    $or: [
      {hash: md5(args.url)},
      {title: args.title}
    ]
  }, {
    $setOnInsert: {
      created_at: new Date().valueOf()
    },
    $set: {
      channel: args.channel,
      section: args.section,
      url: args.url,
      title: args.title,
      description: args.description
    }
  }, {

// Example: go run main.go -test -u http://www.gazeta.pl/0,0.html -p=".mt_list a"
// go run main.go -test -u https://nytimes.com -p=".story a"

https://github.com/asciimoo/colly/blob/master/_examples/hackernews_comments/hackernews_comments.go
https://godoc.org/gopkg.in/mgo.v2#Collection.Find
https://github.com/ungerik/go-dry/blob/master/string.go
https://gobyexample.com