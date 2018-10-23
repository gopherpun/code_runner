package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gopherpun/redis_queue"
	"github.com/sirupsen/logrus"
)

// declares env variables
var (
	RedisHost     string
	ResponseQueue *redis_queue.Queue
	JobQueue      *redis_queue.Queue
	service       string
)

func init() {
	//	logrus.SetFormatter(&logrus.TextFormatter{
	//		TimestampFormat: "2006-01-02T15:04:05.000",
	//		FullTimestamp:   true,
	//	})

	service = "code_runner"

	logrus.SetFormatter(&logrus.JSONFormatter{})

	RedisHost = os.Getenv("REDIS_HOST")
	ResponseQueueKey := os.Getenv("RESPONSE_QUEUE")
	JobQueueKey := os.Getenv("JOB_QUEUE")

	rq, err := redis_queue.NewQueue(RedisHost, ResponseQueueKey)
	if err != nil {
		panic(err)
	}

	jq, err := redis_queue.NewQueue(RedisHost, JobQueueKey)
	if err != nil {
		panic(err)
	}

	ResponseQueue = rq
	JobQueue = jq
}

func main() {
	logrus.WithFields(logrus.Fields{
		"msg":     "Application has started.",
		"service": service,
	}).Info()
	pollQueue()
}

func lambdaRunner(request Job) (events.APIGatewayProxyResponse, error) {
	response := NewBuild(request.Code, request.Language)
	res := encodeResponse(response, request)

	return events.APIGatewayProxyResponse{Headers: map[string]string{}, IsBase64Encoded: false, StatusCode: 200, Body: string(res)}, nil
}

func pollQueue() {
	rate := time.Second
	throttle := time.Tick(rate)
	for {
		<-throttle
		go func() {

			// TODO
			ready, _ := JobQueue.Peek()

			if !ready {
				return
			}

			item, err := JobQueue.Dequeue()
			if err != nil {
				return
			}

			job, _ := decodeJob(item)
			response := NewBuild(job.Code, job.Language)

			res := encodeResponse(response, job)

			ResponseQueue.Enqueue(res)

		}()
	}
}

// Response is a JSON struct represention information about the response.
type Response struct {
	ChannelID string
	Code      string
	Language  string
	RequestID string
	Response  string
}

func encodeResponse(response string, job Job) string {
	jsonJob, _ := json.Marshal(Response{job.ChannelID, job.Code, job.Language, job.RequestID, response})

	return string(jsonJob)
}

// Job is a JSON structure representing information about the job.
type Job struct {
	ChannelID string
	Code      string
	Language  string
	RequestID string
}

func decodeJob(work string) (Job, error) {
	var job Job

	err := json.Unmarshal([]byte(work), &job)

	if err != nil {
		fmt.Println(err)
	}

	return job, err
}
