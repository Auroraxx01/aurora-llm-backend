package main

import (
	"aurora-llm/persistent"
	"aurora-llm/pkg"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	resty "github.com/go-resty/resty/v2"
	"io"
	"log"
	"os"
	"path"
	"sync"
	"time"
)

const (
	fileRoot = "./files"
	filePath = "/files"
)

func resp(c *gin.Context, data any) {
	c.JSON(200, gin.H{"data": data})
}
func fail(c *gin.Context, err error) {
	c.JSON(400, gin.H{"err": err})
}

var fileLock sync.Map

func main() {
	db := persistent.InitDB()
	defer db.Close()
	persistent.InitCollection()

	engine := gin.Default()
	engine.Use(printRequest)
	engine.Static("/dist", "./dist")
	v1 := engine.Group("v1")
	v1.Static(filePath, fileRoot)

	//curl https://api.openai.com/v1/files/file-abc123/content \
	//  -H "Authorization: Bearer $OPENAI_API_KEY" > file.jsonl

	auth := "Bearer " + pkg.AuthToken
	cli := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).SetHeader("Authorization", auth)

	v1.GET("conversations", func(context *gin.Context) {
		threadIDs, err := persistent.GetThreads()
		if err != nil {
			fail(context, err)
			return
		}
		var x []map[string]string
		for i, threadID := range threadIDs {
			x = append(x, map[string]string{"code": threadID, "name": fmt.Sprintf("Thread-%d", i+1)})
		}
		resp(context, x)
	})

	v1.POST("new_conversation", func(c *gin.Context) {
		thread, err := pkg.NewConversation()
		if err != nil {
			fail(c, err)
			return
		}
		_, err = persistent.CreateThread(thread)
		if err != nil {
			resp(c, err)
			return
		}
		resp(c, thread)
	})

	v1.POST("upload", func(c *gin.Context) {
		file, _ := c.FormFile("file")
		filename := file.Filename
		//fullFilePath := path.Join(fileRoot, filename)
		//err := c.SaveUploadedFile(file, fullFilePath)
		openedFile, _ := file.Open()
		defer openedFile.Close()

		blob, err := io.ReadAll(openedFile)
		if err != nil {
			resp(c, err)
			return
		}
		threadID := c.Query("thread_id")
		if threadID == "" {
			fail(c, errors.New("thread_id is required"))
			return
		}
		//response, err := cli.R().SetFile("file", fullFilePath).Post("https://api.openai.com/v1/files")
		uploadFile, err := pkg.UploadFile(filename, blob)
		if err != nil {
			fail(c, err)
			return
		}
		//var uploadFile openai.File
		//err = json.Unmarshal(response.Body(), &uploadFile)
		//if err != nil {
		//	fail(c, err)
		//	return
		//}
		_, err = persistent.SaveUploadedFile(threadID, uploadFile)
		if err != nil {
			fail(c, err)
			return
		}
		resp(c, nil)
	})

	v1.POST("send_message", func(c *gin.Context) {
		threadID := c.PostForm("thread_id")
		if threadID == "" {
			fail(c, errors.New("thread_id is required"))
			return
		}
		msg := c.PostForm("message")
		if msg == "" {
			fail(c, errors.New("message is required"))
			return
		}
		message, run, err := pkg.SendMessage(threadID, msg)
		if err != nil {
			resp(c, err)
			return
		}
		resp(c, map[string]any{"message": message, "run": run})
	})

	v1.GET("messages", func(c *gin.Context) {
		threadID := c.Query("thread_id")
		if threadID == "" {
			fail(c, errors.New("thread_id is required"))
			return
		}
		msgID := c.Query("message_id")
		if msgID == "" {
			messages, err := pkg.ListAllMessage(threadID)
			if err != nil {
				fail(c, err)
				return
			}
			resp(c, messages)
			return
		}
		messages, err := pkg.ListMessage(threadID, msgID)
		if err != nil {
			resp(c, err)
			return
		}
		resp(c, messages)
	})

	v1.POST("retrieve_file", func(c *gin.Context) {
		fileID := c.PostForm("file_id")
		if fileID == "" {
			fail(c, errors.New("file_id is required"))
			return
		}
		fileType := c.PostForm("file_type")
		filename := fileID
		if fileType != "" {
			filename = fileID + ".png"
		}
		fullFilePath := path.Join(fileRoot, filename)
		fileExist := true
		_, err := os.Stat(fullFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				fileExist = false
			}
		}
		if fileExist {
			resp(c, map[string]string{"file_uri": path.Join(filePath, filename)})
			return
		}

		lock, _ := fileLock.LoadOrStore(fileID, &sync.Mutex{})
		lock.(*sync.Mutex).Lock()
		defer lock.(*sync.Mutex).Unlock()

		res, err := cli.R().Get(fmt.Sprintf("https://api.openai.com/v1/files/%s/content", fileID))

		file, err := os.Create(path.Join(fileRoot, filename))
		if err != nil {
			fail(c, err)
			return
		}
		defer file.Close()
		//if _, err = io.Copy(file, readCloser); err != nil {
		//	fail(c, err)
		//	return
		//}
		_, err = file.Write(res.Body())
		if err != nil {
			fail(c, err)
			return
		}

		resp(c, map[string]string{"file_uri": path.Join(filePath, filename)})
	})

	err := engine.Run(":8080")
	if err != nil {
		log.Fatalln(err)
	}
}

// handle access request
func printRequest(ctx *gin.Context) {
	startTime := time.Now()
	// only write body when request json
	log.Printf("path=%v method=%v Content-Type=%v", ctx.FullPath(), ctx.Request.Method, ctx.GetHeader("Content-Type"))
	// log after request executed
	ctx.Next()
	log.Printf("ip: %-15s latency: %-10v code: %-5d method: %-8s path: %s \n",
		ctx.ClientIP(), time.Since(startTime), ctx.Writer.Status(), ctx.Request.Method, ctx.FullPath())
}
