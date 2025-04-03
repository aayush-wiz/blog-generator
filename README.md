# Blog Generator

A full-stack application that automatically generates comprehensive blog posts on any topic using web scraping, AI content generation, and image search.

## 🚀 Features

- **Topic-Based Blog Generation**: Enter any topic to generate a full blog post with relevant content
- **Web Scraping**: Automatically researches topic by scraping latest content from reliable sources
- **Topic-Relevant Images**: Integrates with Pexels API to fetch images that match the blog topic
- **Structured Content**: Creates well-organized blogs with headings, paragraphs, quotes, lists, and images
- **Responsive UI**: Clean, modern React interface to request and view generated blogs

## 🛠️ Tech Stack

### Backend
- **Go**: High-performance API server
- **Colly**: Web scraping framework for content research
- **Mux**: HTTP router for API endpoints
- **CORS**: Cross-origin resource sharing support

### Frontend
- **React**: UI library for the single-page application
- **Vite**: Modern build tool for React development
- **Tailwind CSS**: Utility-first CSS framework for styling

## 📋 API Endpoints

- `POST /api/generate-blog`: Generate a new blog post based on a topic
- `GET /api/blogs`: Retrieve all previously generated blogs
- `GET /api/blogs/{id}`: Get a specific blog by ID
- `GET /api/proxy-image`: Proxy service for fetching external images

## 🔧 Setup

### Prerequisites
- Go 1.16+
- Node.js 16+
- Pexels API key

### Backend Setup
```bash
cd backend
go mod download
# Create .env file with your API keys
echo "PEXELS_API_KEY=your_pexels_key_here" > .env
go run main.go
```

### Frontend Setup
```bash
cd frontend
npm install
npm run dev
```

## 🚦 Usage

1. Start both backend and frontend servers
2. Navigate to http://localhost:5173 in your browser
3. Enter a blog topic in the input field
4. Click "Generate Blog" and wait a few moments
5. View your automatically generated blog post with relevant content and images

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
