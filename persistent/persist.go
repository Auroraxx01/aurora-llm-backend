package persistent

import (
	"fmt"
	c "github.com/ostafen/clover/v2"
	d "github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
)

const (
	databaseName      = "clove-db-aurora"
	collectionThreads = "threads"
	collectionFiles   = "files"
)

var singleDB *c.DB

func InitDB() *c.DB {
	_, err := os.Open(databaseName)
	if err != nil {
		_ = os.Mkdir(databaseName, 0755)
	}
	return getDB()
}

func destroyDB() {
	_ = os.RemoveAll(databaseName)
}

func InitCollection() {
	db := getDB()
	collectionExists, err := db.HasCollection(collectionThreads)
	if err != nil {
		log.Panicf("Failed to check collection: %v", err)
	}
	if !collectionExists {
		if err = db.CreateCollection(collectionThreads); err != nil {
			log.Panicf("Failed to create collection: %v", err)
		}
	}
	collectionExists, err = db.HasCollection(collectionFiles)
	if err != nil {
		log.Panicf("Failed to check collection: %v", err)
	}
	if !collectionExists {
		if err = db.CreateCollection(collectionFiles); err != nil {
			log.Panicf("Failed to create collection: %v", err)
		}
	}
}

func getDB() *c.DB {
	if singleDB == nil {
		db, err := c.Open(databaseName)
		if err != nil {
			panic(err)
		}
		singleDB = db
	}
	return singleDB
}

func CreateThread(thread openai.Thread) (string, error) {
	doc := d.NewDocumentOf(thread)
	docID, err := getDB().InsertOne(collectionThreads, doc)
	if err != nil {
		return "", err
	}
	return docID, nil
}

func GetThreads() ([]string, error) {
	docs, err := getDB().FindAll(query.NewQuery(collectionThreads))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, doc := range docs {
		fmt.Println("????", doc.ToMap())
		tID, ok := doc.Get("ID").(string)
		if ok {
			ids = append(ids, tID)
		}
	}
	return ids, nil
}

func SaveUploadedFile(threadID string, file openai.File) (string, error) {
	doc := d.NewDocumentOf(file)
	doc.Set("thread_id", threadID)
	docID, err := getDB().InsertOne(collectionFiles, doc)
	if err != nil {
		return "", err
	}
	return docID, nil
}

func GetFilesByThreadID(id string) (files []string, err error) {
	q := query.NewQuery(collectionFiles).Where(query.Field("thread_id").Eq(id))
	documents, err := getDB().FindAll(q)
	if err != nil {
		return
	}
	for _, doc := range documents {
		log.Println("get files by thread id:", doc.ToMap())
		fileID, ok := doc.Get("ID").(string)
		if ok {
			files = append(files, fileID)
		}
	}
	return
}
