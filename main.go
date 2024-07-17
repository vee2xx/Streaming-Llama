package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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

	r := gin.Default()
	r.Static("/public", "./public")
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
		})
		fmt.Println(body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		reqBody := bytes.NewBuffer(body)
		req, err = http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", reqBody)
		fmt.Print(req.Body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var openAIResp OpenAIResponse
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Println(openAIResp.Choices[0].Message.Content)

		if len(openAIResp.Choices) > 0 {
			responseText := openAIResp.Choices[0].Message.Content
			history = append(history, PromptRequest{Prompt: responseText})
			c.JSON(http.StatusOK, gin.H{"response": responseText})
		} else {
			myString := string(body[:])
			c.JSON(http.StatusOK, gin.H{"response": myString})
		}

	})

	r.Run(":3000")
}
