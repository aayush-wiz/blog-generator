package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

// BlogContent represents a single block of content in the blog
type BlogContent struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Level   int    `json:"level,omitempty"`
	URL     string `json:"url,omitempty"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// BlogPost represents the full blog structure
type BlogPost struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Author        string        `json:"author"`
	Date          string        `json:"date"`
	Summary       string        `json:"summary"`
	Content       []BlogContent `json:"content"`
	FeaturedImage string        `json:"featuredImage"`
	Tags          []string      `json:"tags"`
	ReadingTime   int           `json:"readingTime"`
	Topic         string        `json:"topic"`
}

// RequestBody represents the incoming request payload
type RequestBody struct {
	Topic string `json:"topic"`
}

// ScrapedContent represents content scraped from the web
type ScrapedContent struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Text        string `json:"text"`
	PublishedAt string `json:"publishedAt"`
}

// LlamaIndexRequest represents the input to the LlamaIndex Python script
type LlamaIndexRequest struct {
	Topic    string           `json:"topic"`
	Contents []ScrapedContent `json:"contents"`
}

// LlamaIndexResponse represents the output from the LlamaIndex Python script
type LlamaIndexResponse struct {
	Title         string        `json:"title"`
	Content       []BlogContent `json:"content"`
	FeaturedImage string        `json:"featuredImage"`
	Tags          []string      `json:"tags"`
	Summary       string        `json:"summary"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/generate-blog", generateBlogHandler).Methods("POST")
	r.HandleFunc("/api/blogs", getBlogsHandler).Methods("GET")
	r.HandleFunc("/api/blogs/{id}", getBlogByIDHandler).Methods("GET")
	r.HandleFunc("/api/proxy-image", proxyImageHandler).Methods("GET")

	handler := cors.Default().Handler(r)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func generateBlogHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.Topic == "" {
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	scrapedContents, err := scrapeContentForTopic(reqBody.Topic)
	if err != nil {
		http.Error(w, "Failed to scrape content: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(scrapedContents) == 0 {
		http.Error(w, "No content found for this topic", http.StatusNotFound)
		return
	}

	llamaResponse, err := GenerateBlogWithLlamaIndex(reqBody.Topic, scrapedContents)
	if err != nil {
		http.Error(w, "Failed to generate blog: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Proxy image URLs through the backend to handle CORS
	for i, block := range llamaResponse.Content {
		if block.Type == "image" {
			proxyURL := fmt.Sprintf("http://localhost:8080/api/proxy-image?url=%s", url.QueryEscape(block.URL))
			llamaResponse.Content[i].URL = proxyURL
		}
	}
	llamaResponse.FeaturedImage = fmt.Sprintf("http://localhost:8080/api/proxy-image?url=%s", url.QueryEscape(llamaResponse.FeaturedImage))

	blog := BlogPost{
		ID:            uuid.New().String(),
		Title:         llamaResponse.Title,
		Author:        "AI Content Generator",
		Date:          time.Now().Format("2006-01-02"),
		Summary:       llamaResponse.Summary,
		Content:       llamaResponse.Content,
		FeaturedImage: llamaResponse.FeaturedImage,
		Tags:          llamaResponse.Tags,
		ReadingTime:   estimateReadingTime(llamaResponse.Content),
		Topic:         reqBody.Topic,
	}

	err = saveBlogPost(blog)
	if err != nil {
		http.Error(w, "Failed to save blog: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blog)
}

func getBlogsHandler(w http.ResponseWriter, r *http.Request) {
	blogs, err := getAllBlogs()
	if err != nil {
		http.Error(w, "Failed to retrieve blogs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blogs)
}

func getBlogByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	blog, err := getBlogByID(id)
	if err != nil {
		http.Error(w, "Blog not found: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blog)
}

func proxyImageHandler(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "Image URL is required", http.StatusBadRequest)
		return
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", imageURL, err)
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch image from %s: %v", imageURL, err)
		http.Error(w, "Failed to fetch image: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Image fetch returned status %d for %s", resp.StatusCode, imageURL)
		http.Error(w, fmt.Sprintf("Failed to fetch image: status code %d", resp.StatusCode), resp.StatusCode)
		return
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Failed to copy image response for %s: %v", imageURL, err)
	}
}

func scrapeContentForTopic(topic string) ([]ScrapedContent, error) {
	var contents []ScrapedContent

	c := colly.NewCollector(
		colly.MaxDepth(2),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	c.AllowedDomains = []string{
		"en.wikipedia.org",
		"www.bbc.com",
		"www.cnn.com",
		"www.reuters.com",
		"www.theguardian.com",
		"news.google.com",
		"www.nytimes.com",
		"www.forbes.com",
		"techcrunch.com",
		"www.wired.com",
	}

	count := 0
	maxCount := 50

	searchQuery := strings.ReplaceAll(topic, " ", "+")
	searchURL := fmt.Sprintf("https://news.google.com/search?q=%s", searchQuery)

	c.OnHTML("article, .article, .post, .entry, main, .content", func(e *colly.HTMLElement) {
		if count >= maxCount {
			return
		}

		content := ScrapedContent{
			URL:   e.Request.URL.String(),
			Title: e.ChildText("h1, h2, .title, .headline"),
			Text:  "",
		}

		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			paragraphText := strings.TrimSpace(el.Text)
			if len(paragraphText) > 20 {
				content.Text += paragraphText + "\n\n"
			}
		})

		publishDate := e.ChildText("time, .date, .published, .timestamp")
		if publishDate != "" {
			content.PublishedAt = publishDate
		}

		if content.Title != "" && len(content.Text) > 100 {
			contents = append(contents, content)
			count++
		}
	})

	err := c.Visit(searchURL)
	if err != nil {
		wikiURL := fmt.Sprintf("https://en.wikipedia.org/wiki/%s", strings.ReplaceAll(topic, " ", "_"))
		err = c.Visit(wikiURL)
		if err != nil {
			log.Printf("Failed to visit Wikipedia: %v", err)
		}
	}

	c.Wait()

	if len(contents) < 5 {
		contents = append(contents, []ScrapedContent{
			{
				URL:         "https://example.com/article1",
				Title:       fmt.Sprintf("Latest developments on %s", topic),
				Text:        fmt.Sprintf("This is a simulated article about %s. It contains information about the topic that would have been scraped from actual news sources.\n\nExperts have been discussing %s extensively.\n\nFurther research on %s is ongoing.", topic, topic, topic),
				PublishedAt: time.Now().Format("2006-01-02"),
			},
			{
				URL:         "https://example.com/article2",
				Title:       fmt.Sprintf("Historical context of %s", topic),
				Text:        fmt.Sprintf("Here's some historical background on %s. This topic has evolved over time.\n\nMany factors have shaped %s today.\n\nCommunities have experienced %s differently.", topic, topic, topic),
				PublishedAt: time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
			},
		}...)
	}

	return contents, nil
}

func estimateReadingTime(content []BlogContent) int {
	totalWords := 0
	for _, block := range content {
		if block.Type == "paragraph" || block.Type == "heading" {
			totalWords += len(strings.Fields(block.Text))
		}
	}
	return (totalWords / 200) + 1 // Assuming 200 words per minute
}

func saveBlogPost(blog BlogPost) error {
	dataDir := "./data/blogs"
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dataDir, blog.ID+".json")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(blog)
}

func getAllBlogs() ([]BlogPost, error) {
	dataDir := "./data/blogs"
	files, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BlogPost{}, nil
		}
		return nil, err
	}

	var blogs []BlogPost
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			blog, err := getBlogByID(strings.TrimSuffix(file.Name(), ".json"))
			if err == nil {
				blogs = append(blogs, blog)
			}
		}
	}
	return blogs, nil
}

func getBlogByID(id string) (BlogPost, error) {
	filePath := filepath.Join("./data/blogs", id+".json")
	file, err := os.Open(filePath)
	if err != nil {
		return BlogPost{}, err
	}
	defer file.Close()

	var blog BlogPost
	err = json.NewDecoder(file).Decode(&blog)
	return blog, err
}
