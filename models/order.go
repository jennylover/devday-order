package models

import (
	"crypto/tls"
	"net"
	"fmt"
	"io/ioutil"

	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Order struct {
	ID        			bson.ObjectId		`json:"id" bson:"_id,omitempty"`
	EmailAddress      string  				`json:"emailAddress"`
	Product           string  				`json:"product"`
	Total             float64 				`json:"total"`
	Status            string  				`json:"status"`
}

// Environment variables
var mongoHost = os.Getenv("MONGOHOST")
var mongoUsername = os.Getenv("MONGOUSER")
var mongoPassword = os.Getenv("MONGOPASSWORD")
var mongoSSL = false 
var mongoPort = ""
var mongoPoolLimit = 25

// MongoDB variables
var mongoDBSession *mgo.Session
var mongoDBSessionError error

// MongoDB database and collection names
var mongoDatabaseName = "hellomongo"
var mongoCollectionName = "orders"
var mongoCollectionShardKey = "_id"

// For tracking and code branching purposes
var isDocDb = strings.Contains(mongoHost, "docdb.amazonaws.com")
var db string // CosmosDB or MongoDB?

// ReadMongoPasswordFromSecret reads the mongo password from the flexvol mount if present
func ReadMongoPasswordFromSecret(file string) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	secret := string(b)
	return secret, err

}

// AddOrderToMongoDB Adds the order to MongoDB/DocumentDB
func AddOrderToMongoDB(order Order) (string, error) {

	// Use the existing mongoDBSessionCopy
	mongoDBSessionCopy := mongoDBSession.Copy()
	defer mongoDBSessionCopy.Close()

	order.ID = bson.NewObjectId()
	StringOrderID := order.ID.Hex()
	order.Status = "Open"

	log.Println("Inserting into MongoDB URL: ", mongoHost, " DocumentDB: ", isDocDb)

	mongoDBCollection := mongoDBSessionCopy.DB(mongoDatabaseName).C(mongoCollectionName)
	mongoDBSessionError = mongoDBCollection.Insert(order)

	if mongoDBSessionError != nil {
		printErr("Problem inserting data: ", mongoDBSessionError)
	} else {
		log.Println("Inserted order:", StringOrderID)
	}

	if(mongoDBSessionError != nil) {
		printErr("MongoDB session error while inserting order: ", mongoDBSessionError.Error())
	}
	return StringOrderID, mongoDBSessionError
}

// GetNumberOfOrdersInDB
func GetNumberOfOrdersInDB() (int, error) {

	mongoDBSessionCopy := mongoDBSession.Copy()
	defer mongoDBSessionCopy.Close()

	log.Println("Querying MongoDB URL: ", mongoHost, " DocumentDB: ", isDocDb)

	mongoDBCollection := mongoDBSessionCopy.DB(mongoDatabaseName).C(mongoCollectionName)
	orderCount,mongoDBSessionError := mongoDBCollection.Count()

	if mongoDBSessionError != nil {
		printErr("Problem quering number of orders: ", mongoDBSessionError)
	} else {
		log.Println("Order count:", orderCount)
	}

	if(mongoDBSessionError != nil) {
		printErr("MongoDB session error while retreiving count: ", mongoDBSessionError.Error())
	}
	return orderCount, mongoDBSessionError
}

//// BEGIN: NON EXPORTED FUNCTIONS
func init() {
	
	log.SetOutput(os.Stdout)

	rand.Seed(time.Now().UnixNano())

	if mongoPassword == "" {
		secret, err := ReadMongoPasswordFromSecret("/kvmnt/mongo-password")
		if err != nil {
			fmt.Print(err)
		}
		mongoPassword = secret
		fmt.Println(mongoPassword)
	}

	validateVariable(mongoHost, "MONGOHOST")
	validateVariable(mongoUsername, "MONGOUSERNAME")
	validateVariable(mongoPassword, "MONGOPASSWORD")

	var mongoPoolLimitEnv = os.Getenv("MONGOPOOL_LIMIT")
	if mongoPoolLimitEnv != "" {
		if limit, err := strconv.Atoi(mongoPoolLimitEnv); err == nil {
			mongoPoolLimit = limit
		}
	}
	log.Printf("MongoDB pool limit set to %v. You can override by setting the MONGOPOOL_LIMIT environment variable." , mongoPoolLimit)

	initMongo()
}

// Logs out value of a variable
func validateVariable(value string, envName string) {
	if len(value) == 0 {
		log.Printf("The environment variable %s has not been set", envName)
	} else {
		log.Printf("The environment variable %s is %s", envName, value)
	}
}

func initMongoDial() (success bool, mErr error) {
	if isDocDb {
		log.Println("Using DocumentDB")
		db = "DocumentDB"
		mongoSSL = true
		mongoPort = ":27017"

	} else {
		log.Println("Using MongoDB")
		db = "MongoDB"
		mongoSSL = false
		mongoPort = ""
	}

	var dialInfo *mgo.DialInfo
	
	mongoDatabase := mongoDatabaseName

	log.Printf("\tUsername: %s", mongoUsername)
	log.Printf("\tPassword: %s", mongoPassword)
	log.Printf("\tHost: %s", mongoHost)
	log.Printf("\tPort: %s", mongoPort)
	log.Printf("\tDatabase: %s", mongoDatabase)
	log.Printf("\tSSL: %t", mongoSSL)

	if mongoSSL {
		dialInfo = &mgo.DialInfo{
			Addrs:    []string{mongoHost+mongoPort},
			Timeout:  10 * time.Second,
			Database: mongoDatabase,
			Username: mongoUsername,
			Password: mongoPassword,
			DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
				return tls.Dial("tcp", addr.String(), &tls.Config{})
			},
		}
	} else {
		dialInfo = &mgo.DialInfo{
			Addrs:    []string{mongoHost+mongoPort},
			Timeout:  10 * time.Second,
			Database: mongoDatabase,
			Username: mongoUsername,
			Password: mongoPassword,
		}
	}

	// Create a mongoDBSession which maintains a pool of socket connections to our MongoDB.
	success = false

	log.Println("Attempting to connect to MongoDB")
	mongoDBSession, mongoDBSessionError = mgo.DialWithInfo(dialInfo)
	if mongoDBSessionError != nil {
		printErr(fmt.Sprintf("Can't connect to mongo at [%s], go error: ", mongoHost+mongoPort), mongoDBSessionError)
		mErr = mongoDBSessionError
	} else {
		success = true
		log.Println("\tConnected")

		mongoDBSession.SetMode(mgo.Monotonic, true)
		
		mongoDBSession.SetPoolLimit(mongoPoolLimit)
	}

	return
}

// Initialize the MongoDB client
func initMongo() {

	success, err := initMongoDial()
	if !success {
		os.Exit(1)
	}

	mongoDBSessionCopy := mongoDBSession.Copy()
	defer mongoDBSessionCopy.Close()

	// SetSafe changes the mongoDBSessionCopy safety mode.
	// If the safe parameter is nil, the mongoDBSessionCopy is put in unsafe mode, and writes become fire-and-forget,
	// without error checking. The unsafe mode is faster since operations won't hold on waiting for a confirmation.
	// http://godoc.org/labix.org/v2/mgo#Session.SetMode.
	mongoDBSessionCopy.SetSafe(nil)

	result := bson.M{}
	err = mongoDBSessionCopy.DB(mongoDatabaseName).Run(
		bson.D{
			{
				"shardCollection",
				fmt.Sprintf("%s.%s", mongoDatabaseName, mongoCollectionName),
			},
			{
				"key",
				bson.M{
					mongoCollectionShardKey: "hashed",
				},
			},
		}, &result)

	if err != nil {
		printErr("Could not create/re-create sharded MongoDB collection. Either collection is already sharded or sharding is not supported. You can ignore this error: ", err)
	} else {
		log.Println("Created MongoDB collection: ")
		log.Println(result)
	}
}

// random: Generates a random number
func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func printErr(v ...interface{}) {
	log.SetOutput(os.Stderr)
	log.Println(v);
	log.SetOutput(os.Stdout)
}

//// END: NON EXPORTED FUNCTIONS
