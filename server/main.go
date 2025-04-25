package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Character struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Creator struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

var characters = []Character{
	{1, "Link", "The hero of Hyrule, a courageous swordsman."},
	{2, "Zelda", "The princess of Hyrule, bearer of the Triforce of Wisdom."},
	{3, "Ganon", "The primary antagonist, seeking to conquer Hyrule."},
}

var creators = []Creator{
	{1, "Shigeru Miyamoto", "Producer"},
	{2, "Eiji Aonuma", "Director"},
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, relying on environment variables")
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/characters", getCharacters)
	r.GET("/characters/search", searchCharacters)
	r.GET("/creators", getCreators)
	r.POST("/chat", chatWithLink)

	r.Run(":8080")
}

func getCharacters(c *gin.Context) {
	c.JSON(http.StatusOK, characters)
}

func getCreators(c *gin.Context) {
	c.JSON(http.StatusOK, creators)
}

func searchCharacters(c *gin.Context) {
	query := strings.ToLower(c.Query("q"))
	var results []Character
	for _, char := range characters {
		if strings.Contains(strings.ToLower(char.Name), query) {
			results = append(results, char)
		}
	}
	c.JSON(http.StatusOK, results)
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func chatWithLink(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY is not set")
	} else {
		log.Println("GEMINI_API_KEY loaded successfully")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-1.5-pro:generateContent?key=%s", apiKey)
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": "You are Link from The Legend of Zelda: Breath of the Wild. Speak courageously, concisely, and with a heroic tone. Avoid modern slang and stay true to the character's personality. User: " + req.Message,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal request"})
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to contact Gemini API"})
		return
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gemini API error: %s, body: %s", resp.Status, string(body))})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read Gemini response"})
		return
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Gemini response"})
		return
	}

	log.Println("Gemini Response Body:", string(body))

	reply := "No reply"
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		reply = geminiResp.Candidates[0].Content.Parts[0].Text
	}

	c.JSON(http.StatusOK, ChatResponse{Reply: reply})
}
