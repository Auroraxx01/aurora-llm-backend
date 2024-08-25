package pkg

import (
	"aurora-llm/persistent"
	"context"
	"encoding/json"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"os"
	"time"
)

var limit100 = 100

var assistantID = func() string {
	id, ok := os.LookupEnv("ASSISTANT_ID")
	if !ok {
		log.Panic("ASSISTANT_ID not set")
	}
	return id
}()

var cli = func() *openai.Client {
	authToken, ok := os.LookupEnv("AUTH_TOKEN")
	if !ok {
		log.Panic("AUTH_TOKEN not set")
	}
	return openai.NewClient(authToken)
}()

var ctxWithTimeout = func(sec int) (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
}

func NewConversation() (openai.Thread, error) {
	ctx, cancel := ctxWithTimeout(10)
	defer cancel()
	threadReq := openai.ThreadRequest{
		Messages:      nil,
		Metadata:      nil,
		ToolResources: nil,
	}

	thread, err := cli.CreateThread(ctx, threadReq)
	if err != nil {
		return thread, err
	}
	return thread, nil
}

func UploadFile(filename string, blob []byte) (openai.File, error) {
	ctx, cancel := ctxWithTimeout(30)
	defer cancel()

	fileBytesReq := openai.FileBytesRequest{
		Name:    filename,
		Bytes:   blob,
		Purpose: openai.PurposeAssistants,
	}
	file, err := cli.CreateFileBytes(ctx, fileBytesReq)
	if err != nil {
		return file, err
	}
	return file, err
}

var messageCodeInterpreter = []openai.Tool{{Type: "code_interpreter"}}

func SendMessage(threadID, msg string) (message openai.Message, run openai.Run, err error) {
	fileIDs, err := persistent.GetFilesByThreadID(threadID)
	if err != nil {
		return
	}
	ctx, cancel := ctxWithTimeout(20)
	defer cancel()

	messageReq := openai.MessageRequest{
		Role:    string(openai.ThreadMessageRoleUser),
		Content: msg,
	}
	for i := range fileIDs {
		messageReq.Attachments = append(messageReq.Attachments, openai.Attachment{FileId: fileIDs[i], Tools: messageCodeInterpreter})
	}
	msgReq, _ := json.Marshal(messageReq)
	log.Println("send message:", string(msgReq))
	message, err = cli.CreateMessage(ctx, threadID, messageReq)
	if err != nil {
		return
	}
	ctx2, cancel2 := ctxWithTimeout(20)
	defer cancel2()

	runReq := openai.RunRequest{
		AssistantID: assistantID,
	}
	run, err = cli.CreateRun(ctx2, threadID, runReq)
	if err != nil {
		return
	}
	return
}

var asc = "asc"

func ListMessage(threadID, msgID string) (messages openai.MessagesList, err error) {
	ctx, cancel := ctxWithTimeout(30)
	defer cancel()

	var msgIDPointer *string
	if msgID != "" {
		msgIDPointer = &msgID
	}
	messages, err = cli.ListMessage(ctx, threadID, nil, &asc, msgIDPointer, nil)
	if err != nil {
		return
	}
	return
}

func ListAllMessage(threadID string) (messages openai.MessagesList, err error) {
	ctx, cancel := ctxWithTimeout(60)
	defer cancel()
	messages, err = cli.ListMessage(ctx, threadID, &limit100, &asc, nil, nil)
	if err != nil {
		return
	}
	return
}

func RetrieveFileIOReaderCloser(fileID string) (content io.ReadCloser, err error) {
	ctx, cancel := ctxWithTimeout(30)
	defer cancel()
	content, err = cli.GetFileContent(ctx, fileID)
	if err != nil {
		return
	}
	return
}

func statusFinished(status openai.RunStatus) bool {
	for _, x := range []openai.RunStatus{openai.RunStatusFailed, openai.RunStatusCompleted, openai.RunStatusIncomplete, openai.RunStatusExpired, openai.RunStatusCancelled} {
		if x == status {
			return true
		}
	}
	return false
}
