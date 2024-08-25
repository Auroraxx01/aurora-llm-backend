package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

func Test_test(t *testing.T) {
	cli := openai.NewClient(AuthToken)
	var req = openai.ChatCompletionRequest{
		Model: openai.GPT4o20240806,
		Tools: []openai.Tool{
			{
				Type:     "",
				Function: nil,
			},
		},
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Help me generate a random correlation heatmap, and visualize it.",
				ToolCalls: []openai.ToolCall{
					{
						Type:     "",
						Function: openai.FunctionCall{},
					},
				},
			},
		},
	}
	resp, err := cli.CreateChatCompletion(context.Background(), req)
	require.NoError(t, err)
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	fmt.Println(string(b))
}

func TestStream(t *testing.T) {
	cli := openai.NewClient(AuthToken)
	var req = openai.ChatCompletionRequest{
		Model: openai.GPT4o20240806,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Help me generate a random correlation heatmap, and visualize it.",
			},
		},
	}
	resp, err := cli.CreateChatCompletionStream(context.Background(), req)
	require.NoError(t, err)
	for {
		r, err := resp.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatalf("error: %v", err)
			}
			break
		}
		b, err := json.Marshal(r)
		fmt.Println(string(b))
	}

}

func TestNewAssistant(t *testing.T) {
	t.Skip("...")
	name := "test"
	desc := "I am a data analysis assistant, I would love to help you with all kinds of data analysis tasks."
	inst := "you are a professional data analyst, good at data visualization, " +
		"you need to analyze any dataset uploaded by users and generate a report/diagram on users' requests." +
		"If you have vector store(from file search tool), you can make use of all those information " +
		"to help you analyze the data and(or) giving out insight solutions most related."
	cli := openai.NewClient(AuthToken)
	resp, err := cli.CreateAssistant(context.Background(), openai.AssistantRequest{
		Model:        openai.GPT4o20240806,
		Name:         &name,
		Description:  &desc,
		Instructions: &inst,
		Tools: []openai.AssistantTool{
			{
				Type:     openai.AssistantToolTypeCodeInterpreter,
				Function: nil,
			},
			{
				Type:     openai.AssistantToolTypeFileSearch,
				Function: nil,
			},
		},
	})
	require.NoError(t, err)
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	fmt.Println(string(b))
}

func TestCheckRun(t *testing.T) {
	runID := "run_IHyw7Xy0CBxA3jzfFlYbRCB5"
	threadID := "thread_11539DgyPK3nQP42UHcEeCgs"
	ctx := context.Background()
	cli := openai.NewClient(AuthToken)
	resp, err := cli.RetrieveRun(ctx, threadID, runID)
	require.NoError(t, err)
	printMsg(resp)
}

func TestAssistantMessage(t *testing.T) {
	ctx := context.Background()
	cli := openai.NewClient(AuthToken)

	threadReq := openai.ThreadRequest{
		Messages:      nil,
		Metadata:      nil,
		ToolResources: nil,
	}

	thread, err := cli.CreateThread(ctx, threadReq)
	require.NoError(t, err)
	fmt.Println("thread id=", thread.ID)

	f, err := os.OpenFile("../files/combined_data_after_feat_eng.csv", os.O_RDONLY, 0644)
	require.NoError(t, err)

	readAll, err := io.ReadAll(f)
	require.NoError(t, err)
	f.Close()
	fmt.Println("read file name=", f.Name())

	fileBytesReq := openai.FileBytesRequest{
		Name:    f.Name(),
		Bytes:   readAll,
		Purpose: openai.PurposeAssistants,
	}
	file, err := cli.CreateFileBytes(ctx, fileBytesReq)
	require.NoError(t, err)
	fmt.Println("uploalded file id=", file.ID)

	messageReq := openai.MessageRequest{
		Role: string(openai.ThreadMessageRoleUser),
		Content: "please read the file just uploaded, y can be either of the column(feel.good social.life having.my.say join.cultural " +
			"join.leisure money.work learning join.spiritual access.enjoy.places avg.qol), " +
			"then all the demography data will be the x, i want to do the significant test first " +
			"then visualize a demographic chart using the significant data.",
		Attachments: []openai.Attachment{
			{
				FileId: file.ID,
				Tools:  []openai.Tool{{Type: "code_interpreter"}},
			},
		},
	}
	message, err := cli.CreateMessage(ctx, thread.ID, messageReq)
	require.NoError(t, err)
	fmt.Println("message id=", message.ID)

	runReq := openai.RunRequest{
		AssistantID: asstID,
	}
	runC, err := cli.CreateRun(ctx, thread.ID, runReq)
	require.NoError(t, err)
	fmt.Println("run id=", runC.ID)

	beginMsgID := message.ID
	currentMsg := message

	for i := 0; ; {
		msg, loopErr := cli.ListMessage(ctx, thread.ID, nil, &asc, &beginMsgID, nil)
		if loopErr != nil {
			t.Fatalf("error: %v", loopErr)
		}
		if len(msg.Messages) > i && len(msg.Messages[i].Content) > 0 {
			currentMsg = msg.Messages[i]
			//currentMsgID = msg.Messages[0].ID
			printMsg(currentMsg.Content)
		}
		run, loopErr := cli.RetrieveRun(ctx, thread.ID, runC.ID)
		require.NoError(t, loopErr)
		if statusFinished(run.Status) {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	msg, err := cli.ListMessage(ctx, thread.ID, nil, &asc, nil, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(msg.Messages) > 0 {
		//currentMsgID = msg.Messages[0].ID
		printMsg(msg.Messages[len(msg.Messages)-1].Content)
	}
	//cli.RetrieveAssistantFile()
}

func printMsg(msg any) {
	b, _ := json.MarshalIndent(msg, "", "  ")
	fmt.Println(string(b))
}

func TestListMessage(t *testing.T) {
	ctx := context.Background()
	cli := openai.NewClient(AuthToken)
	threadID := "thread_hZMW5H5eAEKdo13PDB05UNsC"
	//msgID := "msg_cWFGMttu9qbmUeE6dYcGN4dF"

	msg, loopErr := cli.ListMessage(ctx, threadID, &limit100, &asc, nil, nil)
	if loopErr != nil {
		t.Fatalf("error: %v", loopErr)
	}
	printMsg(msg)
}

func TestRunAMessage(t *testing.T) {
	ctx := context.Background()
	cli := openai.NewClient(AuthToken)
	threadID := "thread_hZMW5H5eAEKdo13PDB05UNsC"

	// named 'combined_data_after_feat_eng 2.csv'
	//messageReq := openai.MessageRequest{
	//	Role:    string(openai.ThreadMessageRoleUser),
	//	Content: "please visualize a demographic chart, with what you think is significant.",
	//	Attachments: []openai.Attachment{
	//		{
	//			FileId: "file-Aopzqw6MvwsQMAjF3JcxGHm5",
	//			Tools:  []openai.Tool{{Type: "code_interpreter"}},
	//		},
	//	},
	//	Metadata: nil,
	//}
	//message, err := cli.CreateMessage(ctx, threadID, messageReq)
	//require.NoError(t, err)
	//fmt.Println("message id=", message.ID)

	runReq := openai.RunRequest{
		AssistantID: asstID,
	}
	runC, err := cli.CreateRun(ctx, threadID, runReq)
	require.NoError(t, err)
	fmt.Println("run id=", runC.ID)

	beginMsgID := "msg_dLbwKLZxjb3cbceebgGKJS92"
	currentMsg := openai.Message{}

	for i := 0; ; {
		msg, loopErr := cli.ListMessage(ctx, threadID, nil, &asc, &beginMsgID, nil)
		if loopErr != nil {
			t.Fatalf("error: %v", loopErr)
		}
		if len(msg.Messages) > i && len(msg.Messages[i].Content) > 0 {
			currentMsg = msg.Messages[i]
			//currentMsgID = msg.Messages[0].ID
			printMsg(currentMsg.Content)
		}
		run, loopErr := cli.RetrieveRun(ctx, threadID, runC.ID)
		require.NoError(t, loopErr)
		if statusFinished(run.Status) {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	msg, err := cli.ListMessage(ctx, threadID, nil, &asc, nil, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(msg.Messages) > 0 {
		printMsg(msg.Messages[len(msg.Messages)-1].Content)
	}
}

func TestFormat(t *testing.T) {
	str := "The ANOVA results indicate that the following demographic predictors are significantly related to the target variable \"avg.qol\" at a significance level of 0.05:\n\n1. **dis_type** (p-value = 2.04e-08)\n2. **long_term_issues** (p-value = 1.05e-03)\n3. **pri_caregiver** (p-value = 7.62e-06)\n4. **age** (p-value = 1.30e-36)\n\nThese variables are worth keeping for further analysis and model building in machine learning since they show a statistically significant relationship with the quality of life measure (`avg.qol`). Would you like to proceed with any specific analysis using these variables?"
	replacer := strings.NewReplacer("\\n", "\n", "\\", "")
	str = replacer.Replace(str)
	fmt.Println(str)
}

func TestRunAMessage2(t *testing.T) {
	ctx := context.Background()
	cli := openai.NewClient(AuthToken)
	threadID := "thread_11539DgyPK3nQP42UHcEeCgs"

	// named 'combined_data_after_feat_eng 2.csv'  msg_SN3nat8mXZTrnu5YDYuJPDFW???notfound
	//messageReq := openai.MessageRequest{
	//	Role:    string(openai.ThreadMessageRoleUser),
	//	Content: "avg.qol",
	//	Attachments: []openai.Attachment{
	//		{
	//			FileId: "file-TSl6nQoIVxnwLaABEpgPVSoe",
	//			Tools:  []openai.Tool{{Type: "code_interpreter"}},
	//		},
	//		{
	//			FileId: "file-xcDcSS5dAKY1FMwzt8MhH1QJ",
	//			Tools:  []openai.Tool{{Type: "code_interpreter"}},
	//		},
	//	},
	//	Metadata: nil,
	//}
	//message, err := cli.CreateMessage(ctx, threadID, messageReq)
	//require.NoError(t, err)
	//fmt.Println("message id=", message.ID)
	msgId := "msg_nKuWJqKiAPPbAYFIYBiK6lIb"

	runReq := openai.RunRequest{
		AssistantID: asstID,
	}
	runC, err := cli.CreateRun(ctx, threadID, runReq)
	require.NoError(t, err)
	fmt.Println("run id=", runC.ID)

	currentMsgID := msgId
	limit1 := 1
	currentMsg := openai.Message{}
	for {
		msg, loopErr := cli.ListMessage(ctx, threadID, &limit1, nil, nil, &currentMsgID)
		if loopErr != nil {
			t.Fatalf("error: %v", loopErr)
		}
		if len(msg.Messages) > 0 {
			currentMsg = msg.Messages[0]
			currentMsgID = msg.Messages[0].ID
			printMsg(currentMsg)
		}
		run, loopErr := cli.RetrieveRun(ctx, threadID, runC.ID)
		require.NoError(t, loopErr)
		if statusFinished(run.Status) {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	msg, err := cli.ListMessage(ctx, threadID, &limit1, nil, nil, &currentMsgID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(msg.Messages) > 0 {
		currentMsg = msg.Messages[0]
		currentMsgID = msg.Messages[0].ID
		printMsg(currentMsg)
	}
}
