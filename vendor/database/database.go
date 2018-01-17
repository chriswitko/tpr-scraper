package database

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	mgo "gopkg.in/mgo.v2"
)

// Debug enable logger to the console
var Debug bool

// MongoConnection as a globel session
type MongoConnection struct {
	originalSession *mgo.Session
	databaseName    *string
}

func getEnv(key string, isRequired bool) string {
	value := os.Getenv(key)
	if isRequired == true && value == "" {
		panic("Missing setting in .env file")
	}
	return value
}

// CreateConnection connects to database
func (c *MongoConnection) CreateConnection() (err error) {
	dbURI := getEnv("DB_URI", true)
	dbNAME := getEnv("DB_NAME", true)

	c.databaseName = &dbNAME

	fmt.Println("Connecting to local mongo server....")
	// if Debug {
	// log.Println("DB_URI", dbURI)
	// }

	dbURI = strings.TrimSuffix(dbURI, "?ssl=true")
	tlsConfig := &tls.Config{}
	tlsConfig.InsecureSkipVerify = true

	dialInfo, err := mgo.ParseURL(dbURI)
	if err != nil {
		fmt.Println("Failed to parse URI: ", err)
		os.Exit(1)
	}

	dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
		conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
		return conn, err
	}

	c.originalSession, err = mgo.DialWithInfo(dialInfo)

	if err == nil {
		c.originalSession.SetMode(mgo.Monotonic, true)
		fmt.Println("Connection established to mongo server")
		// urlcollection := c.originalSession.DB("LinkShortnerDB").C("UrlCollection")
		// if urlcollection == nil {
		// err = errors.New("Collection could not be created, maybe need to create it manually")
		// }
		//This will create a unique index to ensure that there won't be duplicate shorturls in the database.
		// index := mgo.Index{
		// 	Key:      []string{"$text:shorturl"},
		// 	Unique:   true,
		// 	DropDups: true,
		// }
		// urlcollection.EnsureIndex(index)
	} else {
		fmt.Printf("Error occured while creating mongodb connection: %s", err.Error())
	}
	return
}

// GetSession get the current session and make a copy
func (c *MongoConnection) GetSession() (session *mgo.Session, databaseName string, err error) {
	if c.originalSession != nil {
		session = c.originalSession.Copy()
		databaseName = *c.databaseName
	} else {
		err = errors.New("No original session found")
	}
	return
}

// CloseSession close the current session
func (c *MongoConnection) CloseSession() (session *mgo.Session, err error) {
	if c.originalSession != nil {
		c.originalSession.Close()
		fmt.Println("Database session closed")
	} else {
		err = errors.New("No original session found")
	}
	return
}
