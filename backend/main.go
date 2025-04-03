package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

// BlogContent represents a content block in the blog
type BlogContent struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	Level    int      `json:"level,omitempty"`
	URL      string   `json:"url,omitempty"`
	Alt      string   `json:"alt,omitempty"`
	Caption  string   `json:"caption,omitempty"`
	Citation string   `json:"citation,omitempty"`
	Items    []string `json:"items,omitempty"`
	Ordered  bool     `json:"ordered,omitempty"`
}

// BlogPost structure for blog data
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

// RequestBody for the generate-blog endpoint
type RequestBody struct {
	Topic string `json:"topic"`
}

// ScrapedContent stores content from web scraping
type ScrapedContent struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Text        string `json:"text"`
	PublishedAt string `json:"publishedAt,omitempty"`
}

// LlamaIndexPrompt structure
type LlamaIndexPrompt struct {
	Topic    string           `json:"topic"`
	Contents []ScrapedContent `json:"contents"`
}

// PexelsResponse represents the response from Pexels API
type PexelsResponse struct {
	Photos []struct {
		Src struct {
			Medium string `json:"medium"`
			Large  string `json:"large"`
		} `json:"src"`
	} `json:"photos"`
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/api/generate-blog", generateBlogHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/blogs", getBlogsHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/blogs/{id}", getBlogByIDHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/proxy-image", proxyImageHandler).Methods("GET")

	// Create data directory if it doesn't exist
	os.MkdirAll("data/blogs", os.ModePerm)

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Wrap router with CORS middleware
	handler := c.Handler(r)

	// Start server
	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func generateBlogHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
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

	// 1. Web scraping phase
	scrapedContents, err := scrapeContentForTopic(reqBody.Topic)
	if err != nil {
		http.Error(w, "Failed to scrape content: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(scrapedContents) == 0 {
		http.Error(w, "No content found for this topic", http.StatusNotFound)
		return
	}

	// 2. Index with LlamaIndex (simulated)
	promptData := LlamaIndexPrompt{
		Topic:    reqBody.Topic,
		Contents: scrapedContents,
	}

	// 3. Generate blog post using the indexed content (simulated LLM call)
	blog, err := generateBlogPost(promptData)
	if err != nil {
		http.Error(w, "Failed to generate blog: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Store the generated blog post
	err = saveBlogPost(blog)
	if err != nil {
		http.Error(w, "Failed to save blog: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the generated blog post
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blog)
}

func getBlogsHandler(w http.ResponseWriter, r *http.Request) {
	blogs, err := getAllBlogs()
	if err != nil {
		http.Error(w, "Failed to get blogs: "+err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Blog not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blog)
}

// Web scraping function using Colly
func scrapeContentForTopic(topic string) ([]ScrapedContent, error) {
	var contents []ScrapedContent

	// Initialize Colly collector
	c := colly.NewCollector(
		colly.MaxDepth(2),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	// Limit crawling to relevant domains
	c.AllowedDomains = []string{
		"en.wikipedia.org",
		"www.bbc.com",
		"www.cnn.com",
		"www.reuters.com",
		"www.theguardian.com",
		"news.google.com",
	}

	// Setup a limit counter
	count := 0
	maxCount := 5 // Limit to 5 sources to avoid overwhelming

	// Search query
	searchQuery := strings.ReplaceAll(topic, " ", "+")
	searchURL := fmt.Sprintf("https://news.google.com/search?q=%s", searchQuery)

	// Define scraping callbacks
	c.OnHTML("article, .article, .post, .entry, main, .content", func(e *colly.HTMLElement) {
		if count >= maxCount {
			return
		}

		content := ScrapedContent{
			URL:   e.Request.URL.String(),
			Title: e.ChildText("h1, h2, .title, .headline"),
			Text:  "",
		}

		// Extract paragraphs
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			paragraphText := strings.TrimSpace(el.Text)
			if len(paragraphText) > 20 { // Avoid empty or very short paragraphs
				content.Text += paragraphText + "\n\n"
			}
		})

		// Extract publish date if available
		publishDate := e.ChildText("time, .date, .published, .timestamp")
		if publishDate != "" {
			content.PublishedAt = publishDate
		}

		// Only add if there's meaningful content
		if content.Title != "" && len(content.Text) > 100 {
			contents = append(contents, content)
			count++
		}
	})

	// Visit the search page and follow links
	err := c.Visit(searchURL)
	if err != nil {
		// Fall back to Wikipedia if news search fails
		wikiURL := fmt.Sprintf("https://en.wikipedia.org/wiki/%s", strings.ReplaceAll(topic, " ", "_"))
		err = c.Visit(wikiURL)
		if err != nil {
			return contents, nil // Return whatever we have, even if empty
		}
	}

	// Wait for scraping to finish
	c.Wait()

	// Simulate scraping for development purposes if we didn't get any real content
	if len(contents) == 0 {
		contents = []ScrapedContent{
			{
				URL:         "https://example.com/article1",
				Title:       fmt.Sprintf("Latest developments on %s", topic),
				Text:        fmt.Sprintf("This is a simulated article about %s. It contains information about the topic that would have been scraped from actual news sources. The content includes various aspects and perspectives on the subject matter.\n\nExperts have been discussing the implications of recent developments related to %s. Some argue that it represents a significant shift in how we understand the topic.\n\nFurther research is ongoing to fully understand the nuances of %s and its impact on various sectors.", topic, topic, topic),
				PublishedAt: time.Now().Format("2006-01-02"),
			},
			{
				URL:         "https://example.com/article2",
				Title:       fmt.Sprintf("Historical context of %s", topic),
				Text:        fmt.Sprintf("Here's some historical background on %s. This topic has evolved significantly over the past decade.\n\nMany factors have contributed to how %s is perceived today. Economic, social, and political elements all play a role in shaping the discourse.\n\nCommunities worldwide have experienced the effects of %s in different ways. Their stories provide valuable insights into the global impact of this subject.", topic, topic, topic),
				PublishedAt: time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
			},
		}
	}

	return contents, nil
}

// Function to get topic-related images from Pexels
func getTopicImage(topic string) string {
	// Default fallback image
	defaultImage := "https://cdn.pixabay.com/photo/2018/01/12/10/19/fantasy-3077928_1280.jpg"
	
	// Get API key from environment variables
	apiKey := os.Getenv("PEXELS_API_KEY")
	
	// If no API key is provided, return default image
	if apiKey == "" {
		log.Println("Warning: PEXELS_API_KEY not found in environment")
		return defaultImage
	}
	
	// Create the Pexels API request
	baseURL := "https://api.pexels.com/v1/search"
	queryParams := url.Values{}
	queryParams.Add("query", topic)
	queryParams.Add("per_page", "1")
	
	req, err := http.NewRequest("GET", baseURL+"?"+queryParams.Encode(), nil)
	if err != nil {
		fmt.Printf("Error creating image request: %v\n", err)
		return defaultImage
	}
	
	// Add API key
	req.Header.Set("Authorization", apiKey)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching image: %v\n", err)
		return defaultImage
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error status code: %d\n", resp.StatusCode)
		return defaultImage
	}
	
	// Parse response
	var pexelsResp PexelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&pexelsResp); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return defaultImage
	}
	
	// If photos found, return the first one
	if len(pexelsResp.Photos) > 0 {
		return pexelsResp.Photos[0].Src.Large
	}
	
	return defaultImage
}

// Function to generate blog post (simulating LlamaIndex + LLM integration)
func generateBlogPost(prompt LlamaIndexPrompt) (BlogPost, error) {
	// Generate a unique ID
	blogID := uuid.New().String()
	
	// Get topic images
	featuredImageURL := getTopicImage(prompt.Topic)
	contentImage1 := getTopicImage(prompt.Topic + " overview")
	contentImage2 := getTopicImage(prompt.Topic + " detail")
	
	// Create the blog structure
	blog := BlogPost{
		ID:            blogID,
		Title:         fmt.Sprintf("Comprehensive Guide to %s: Latest Insights and Developments", prompt.Topic),
		Author:        "AI Content Generator",
		Date:          time.Now().Format("2006-01-02"),
		Topic:         prompt.Topic,
		ReadingTime:   5, // Estimated reading time
		FeaturedImage: featuredImageURL,
		Tags:          []string{prompt.Topic, "News", "Analysis"},
	}

	// Generate summary
	blog.Summary = fmt.Sprintf("An in-depth look at %s, examining the latest developments, key insights, and future implications based on the most current information available.", prompt.Topic)

	// Generate content with topic-relevant images
	blog.Content = []BlogContent{
		{Type: "heading", Level: 1, Text: fmt.Sprintf("Understanding %s: A Comprehensive Overview", prompt.Topic)},
		{Type: "paragraph", Text: fmt.Sprintf("In recent times, %s has become an increasingly important topic in global discourse. This article aims to provide a thorough analysis based on the latest information available from reliable sources.", prompt.Topic)},
		{Type: "image", URL: contentImage1, Alt: fmt.Sprintf("Illustration of %s", prompt.Topic), Caption: fmt.Sprintf("Visual representation of key aspects related to %s", prompt.Topic)},
		{Type: "heading", Level: 2, Text: "Current State and Recent Developments"},
	}

	// Add content based on scraped data
	if len(prompt.Contents) > 0 {
		paragraphs := strings.Split(prompt.Contents[0].Text, "\n\n")
		for _, p := range paragraphs[:minInt(3, len(paragraphs))] {
			if len(p) > 0 {
				blog.Content = append(blog.Content, BlogContent{Type: "paragraph", Text: p})
			}
		}
	}

	// Add more structured content with a topic-relevant image
	blog.Content = append(blog.Content, []BlogContent{
		{Type: "heading", Level: 2, Text: "Key Factors and Analysis"},
		{Type: "paragraph", Text: fmt.Sprintf("Several key factors influence the current state of %s. Understanding these elements is crucial for a comprehensive perspective.", prompt.Topic)},
		{Type: "list", Items: []string{
			fmt.Sprintf("Economic implications of %s on global markets", prompt.Topic),
			fmt.Sprintf("Social and cultural impact of %s on communities", prompt.Topic),
			fmt.Sprintf("Technological advancements related to %s", prompt.Topic),
			fmt.Sprintf("Regulatory frameworks surrounding %s", prompt.Topic),
		}, Ordered: false},
		{Type: "image", URL: contentImage2, Alt: fmt.Sprintf("Impact of %s", prompt.Topic), Caption: fmt.Sprintf("Visualizing the multifaceted impact of %s", prompt.Topic)},
		{Type: "heading", Level: 2, Text: "Expert Opinions and Insights"},
		{Type: "quote", Text: fmt.Sprintf("The development of %s represents one of the most significant shifts in this field that we've seen in the past decade.", prompt.Topic), Citation: "Industry Expert"},
	}...)

	return blog, nil
}

// Helper function to save blog post to JSON file
func saveBlogPost(blog BlogPost) error {
	// Create blogs directory if it doesn't exist
	os.MkdirAll("data/blogs", os.ModePerm)

	// Convert to JSON
	blogJSON, err := json.MarshalIndent(blog, "", "  ")
	if err != nil {
		return err
	}

	// Save to file
	filePath := filepath.Join("data/blogs", blog.ID+".json")
	return ioutil.WriteFile(filePath, blogJSON, 0644)
}

// Then add the handler function
func proxyImageHandler(w http.ResponseWriter, r *http.Request) {
    imageURL := r.URL.Query().Get("url")
    if imageURL == "" {
        http.Error(w, "Image URL is required", http.StatusBadRequest)
        return
    }

    fmt.Printf("Proxying image: %s\n", imageURL) // Add logging

    client := &http.Client{
        Timeout: 15 * time.Second,
    }
    
    req, err := http.NewRequest("GET", imageURL, nil)
    if err != nil {
        http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
        fmt.Printf("Error creating request: %v\n", err) // Log error
        return
    }
    
    // Add a proper user agent
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
    
    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, "Failed to fetch image: "+err.Error(), http.StatusInternalServerError)
        fmt.Printf("Error fetching image: %v\n", err) // Log error
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        http.Error(w, fmt.Sprintf("Failed to fetch image: status code %d", resp.StatusCode), resp.StatusCode)
        fmt.Printf("Error status code: %d\n", resp.StatusCode) // Log status code
        return
    }

    // Copy headers
    for k, v := range resp.Header {
        w.Header()[k] = v
    }
    
    // Set content type
    w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
    
    // Copy the body
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}

// Helper function to get all blogs
func getAllBlogs() ([]BlogPost, error) {
	var blogs []BlogPost

	files, err := ioutil.ReadDir("data/blogs")
	if err != nil {
		return blogs, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			blogData, err := ioutil.ReadFile(filepath.Join("data/blogs", file.Name()))
			if err != nil {
				continue
			}

			var blog BlogPost
			if err := json.Unmarshal(blogData, &blog); err != nil {
				continue
			}

			blogs = append(blogs, blog)
		}
	}

	return blogs, nil
}

// Helper function to get blog by ID
func getBlogByID(id string) (BlogPost, error) {
	var blog BlogPost

	filePath := filepath.Join("data/blogs", id+".json")
	blogData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return blog, err
	}

	if err := json.Unmarshal(blogData, &blog); err != nil {
		return blog, err
	}

	return blog, nil
}

// Helper function to get minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
