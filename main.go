// This is a Server Side Event (SSE) Example using Gin Framework.
// In this example, data is sent to the first client on subsequent successful connections.
// After running this program, open up 2 or more tabs of localhost:8085,
// Watch a message getting sent to the first tab (first client).

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type OpenAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type OpenAIRequest struct {
	Model     string          `json:"model"`
	Messages  []OpenAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type UserPrompt struct {
	Prompt string `json:"prompt"`
}

var history []OpenAIMessage

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set in .env file")
	}

	chanStream := make(chan string)
	defer close(chanStream)

	history = append(history, OpenAIMessage{Role: "system", Content: "You are an assistant well versed in general knowledge."})

	//TODO: Make the role configurable

	r := gin.Default()
	r.Static("/public", "./public")
	r.StaticFile("/friendly_llama.jpg", "./public/friendly_llama.jpg")
	r.LoadHTMLFiles("public/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.POST("/api/prompt", func(c *gin.Context) {

		var req *http.Request
		var prompt UserPrompt

		if err := c.BindJSON(&prompt); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		history = append(history, OpenAIMessage{Role: "user", Content: prompt.Prompt})
		var openAIRequest OpenAIRequest = OpenAIRequest{Model: "gpt-3.5-turbo-16k", Messages: history, MaxTokens: 150, Stream: true}

		body, err := json.Marshal(openAIRequest)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		reqBody := bytes.NewBuffer(body)
		req, err = http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", reqBody)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)

		fullResp := ""
		// var tempBuffer strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			// fmt.Println(line)
			if len(line) > 0 {
				if strings.HasPrefix(line, "data: ") {
					line = strings.TrimPrefix(line, "data: ")
				}
				if line != "[DONE]" {
					var openAIResp OpenAIStreamResponse
					if err := json.Unmarshal([]byte(line), &openAIResp); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}
					if len(openAIResp.Choices) > 0 {
						respChunk := openAIResp.Choices[0].Delta.Content
						chanStream <- respChunk
						// tempBuffer.WriteString(respChunk)
						// if strings.HasSuffix(tempBuffer.String(), " ") || strings.HasSuffix(tempBuffer.String(), "\n") {
						// 	chanStream <- tempBuffer.String()
						// 	tempBuffer.Reset()
						// }
						fullResp += respChunk
					}
				} else {
					chanStream <- "[DONE]"
				}

			}
		}

		history = append(history, OpenAIMessage{Role: "assistant", Content: fullResp})
		if err := scanner.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	})

	// SSE endpoint that the clients will be listening to
	r.GET("/stream", func(c *gin.Context) {
		// Set the response header to indicate SSE content type
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		c.Stream(func(w io.Writer) bool {
			if msg, ok := <-chanStream; ok {
				c.SSEvent("message", msg)
				return true
			}
			return false

		})
	})

	r.Run(":3000")
}
