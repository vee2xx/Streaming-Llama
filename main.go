// This is a Server Side Event (SSE) Example using Gin Framework.
// In this example, data is sent to the first client on subsequent successful connections.
// After running this program, open up 2 or more tabs of localhost:8085,
// Watch a message getting sent to the first tab (first client).

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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

type PromptRequest struct {
	Prompt string `json:"prompt"`
}

var history []PromptRequest

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set in .env file")
	}

	chanStream := make(chan string, 10)
	defer close(chanStream)

	r := gin.Default()
	r.Static("/public", "./public")
	r.StaticFile("/friendly_llama.jpg", "./public/friendly_llama.jpg")
	r.LoadHTMLFiles("public/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.POST("/api/prompt", func(c *gin.Context) {
		var preq PromptRequest
		var req *http.Request
		if err := c.BindJSON(&preq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		history = append(history, preq)

		context := ""
		for _, h := range history {
			context += h.Prompt + "\n" + h.Prompt + "\n"
		}

		body, err := json.Marshal(map[string]interface{}{
			"model":      "gpt-3.5-turbo-16k",
			"messages":   []interface{}{map[string]interface{}{"role": "user", "content": context}},
			"max_tokens": 150,
			"stream":     true,
		})

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

		// err = handleStreamResponse(c, resp.Body)
		scanner := bufio.NewScanner(resp.Body)

		tempResp := ""

		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
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
						//ToDo: Need to handle spaces and punctuation, etc.
						//Also, handle 'done'
						respChunk := openAIResp.Choices[0].Delta.Content + " "
						tempResp += respChunk
						chanStream <- respChunk
					} else {
						fmt.Println(tempResp)
					}
				}

			}
		}

		if err := scanner.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// responseText, err := handleResponse(resp.Body)

		// history = append(history, PromptRequest{Prompt: responseText})

		// c.JSON(http.StatusOK, gin.H{"response": responseText})
	})

	// SSE endpoint that the clients will be listening to
	r.GET("/stream", func(c *gin.Context) {
		// Set the response header to indicate SSE content type
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Allow all origins to access the endpoint (Else you will get CORS error)
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

	// Parse Static files
	// r.StaticFile("/", "./index.html")

	r.Run(":3000")
}
