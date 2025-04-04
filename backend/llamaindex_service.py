import sys
import json
import os
import requests
from typing import List, Dict, Any, Optional
from datetime import datetime
import chromadb
from llama_index.core import VectorStoreIndex, Document, StorageContext
from llama_index.vector_stores.chroma import ChromaVectorStore
from llama_index.llms.openai import OpenAI

class LlamaIndexService:
    def __init__(self, api_key: Optional[str] = None):
        self.api_key = api_key or os.environ.get("OPENAI_API_KEY")
        if not self.api_key:
            raise ValueError("OPENAI_API_KEY environment variable is required")
        self.pexels_api_key = os.environ.get("PEXELS_API_KEY")
        if not self.pexels_api_key:
            raise ValueError("PEXELS_API_KEY environment variable is required")
        self.llm = OpenAI(api_key=self.api_key, model="gpt-4")

    def create_documents_from_scraped_content(self, content_list: List[Dict[str, Any]]) -> List[Document]:
        documents = []
        for content in content_list:
            doc = Document(
                text=content.get("text", ""),
                metadata={
                    "source": content.get("url", ""),
                    "title": content.get("title", ""),
                    "date": content.get("publishedAt", datetime.now().isoformat()),
                }
            )
            documents.append(doc)
        return documents

    def create_index(self, documents: List[Document], persist_dir: str = "./data/index") -> VectorStoreIndex:
        chroma_client = chromadb.PersistentClient(path=persist_dir)
        chroma_collection = chroma_client.get_or_create_collection("scraped_content")
        vector_store = ChromaVectorStore(chroma_collection=chroma_collection)
        storage_context = StorageContext.from_defaults(vector_store=vector_store)
        index = VectorStoreIndex.from_documents(documents, storage_context=storage_context)
        return index

    def fetch_pexels_images(self, topic: str, count: int = 2) -> List[str]:
        url = "https://api.pexels.com/v1/search"
        headers = {"Authorization": self.pexels_api_key}
        params = {
            "query": topic,
            "per_page": count,
            "orientation": "landscape",
            "size": "large"
        }
        response = requests.get(url, headers=headers, params=params)
        if response.status_code != 200:
            print(f"Failed to fetch Pexels images: {response.status_code} - {response.text}")
            return [f"https://via.placeholder.com/900x500?text={topic}+Image+Not+Available"] * count
        
        data = response.json()
        photos = data.get("photos", [])
        if not photos:
            print(f"No Pexels images found for topic: {topic}")
            return [f"https://via.placeholder.com/900x500?text={topic}+Image+Not+Available"] * count
        
        return [photo["src"]["large"] for photo in photos[:count]] + [f"https://via.placeholder.com/900x500?text={topic}+Image+Not+Available"] * (count - len(photos))

    def generate_blog_from_query(self, topic: str, index: VectorStoreIndex) -> Dict:
        query_engine = index.as_query_engine(
            llm=self.llm,
            similarity_top_k=20,
            response_mode="tree_summarize"
        )

        prompt = f"""
        Write a comprehensive and engaging blog post about '{topic}' that reads like a professional article.
        Use the retrieved information from the indexed web content to create factual, informative, and reader-friendly content.
        Structure the blog as follows:
        - **Introduction**: 3-5 paragraphs (about 100-150 lines total) that grab attention with a hook (e.g., a question, anecdote, or surprising fact), provide context, and preview the main sections.
        - **Main Body**: 3-5 sections with descriptive headings (e.g., 'What is {topic}?', 'Key Developments', 'Impact on Society', 'Future Prospects'), each with 4-6 paragraphs (about 100-150 lines per section). Include relevant examples, case studies, or anecdotes from the data, and pose questions to engage readers (e.g., 'Have you noticed this in your life?').
        - **Conclusion**: 2-3 paragraphs (about 50-75 lines) summarizing key points and ending with a thought-provoking statement or call to action.
        Include exactly 2 image placeholders: one as the featured image and one in the body after the introduction. Use placeholders like 'FEATURED_IMAGE_URL' and 'CONTENT_IMAGE_URL'; actual URLs will be filled in later.
        Format the response as a JSON object with 'title', 'content' (list of content blocks), 'featuredImage', 'tags', and 'summary'.
        Each content block should have 'type' (e.g., 'heading', 'paragraph', 'image') and appropriate fields (e.g., 'text' for paragraphs, 'url', 'alt', 'caption' for images).
        Ensure the tone is conversational, the content is well-organized, and the output feels like a blog post, not a list of facts or images.
        """

        response = query_engine.query(prompt)
        response_text = str(response).strip()

        # Fetch exactly 2 images from Pexels
        pexels_images = self.fetch_pexels_images(topic, 2)
        featured_image = pexels_images[0] if pexels_images else "https://via.placeholder.com/1200x600?text=Featured+Image+Not+Available"
        content_image = pexels_images[1] if len(pexels_images) > 1 else "https://via.placeholder.com/900x500?text=Content+Image+Not+Available"

        try:
            blog_data = json.loads(response_text)
            if isinstance(blog_data, dict) and "content" in blog_data:
                content_blocks = []
                image_count = 0
                for block in blog_data.get("content", []):
                    if isinstance(block, dict) and "type" in block:
                        if block["type"] == "image":
                            if image_count == 0:
                                block["url"] = featured_image
                                block["alt"] = f"Featured image for {topic}"
                                block["caption"] = f"{topic} Overview"
                            elif image_count == 1:
                                block["url"] = content_image
                                block["alt"] = f"Visual representation of {topic}"
                                block["caption"] = f"Exploring {topic}"
                            image_count += 1
                            if image_count > 2:  # Skip extra images beyond 2
                                continue
                        elif block["type"] == "heading":
                            block.setdefault("level", 1)
                            block.setdefault("text", block.get("content", ""))
                        elif block["type"] == "text":
                            block = {"type": "paragraph", "text": block.get("content", "")}
                        content_blocks.append(block)

                # Ensure exactly 2 images: featured at start, content after intro
                has_featured = any(b["type"] == "image" and b["caption"].startswith(f"{topic} Overview") for b in content_blocks)
                has_content = any(b["type"] == "image" and b["caption"].startswith(f"Exploring {topic}") for b in content_blocks)
                intro_end = 0
                for i, block in enumerate(content_blocks):
                    if block["type"] == "heading" and i > 0:
                        intro_end = i
                        break

                if not has_featured:
                    content_blocks.insert(0, {
                        "type": "image",
                        "url": featured_image,
                        "alt": f"Featured image for {topic}",
                        "caption": f"{topic} Overview"
                    })
                if not has_content and intro_end > 0:
                    content_blocks.insert(intro_end, {
                        "type": "image",
                        "url": content_image,
                        "alt": f"Visual representation of {topic}",
                        "caption": f"Exploring {topic}"
                    })

                blog = {
                    "title": blog_data.get("title", f"Exploring {topic}: A Deep Dive"),
                    "content": content_blocks,
                    "featuredImage": featured_image,
                    "tags": blog_data.get("tags", [topic, "Insights", "Overview"]),
                    "summary": blog_data.get("summary", f"A comprehensive blog post exploring {topic}, its developments, impacts, and future possibilities.")
                }
                return blog
        except json.JSONDecodeError:
            content_blocks = []
            lines = response_text.split("\n")
            current_section = None
            image_count = 0

            for line in lines:
                line = line.strip()
                if not line:
                    continue
                if line.startswith("# "):
                    current_section = {"type": "heading", "level": 1, "text": line[2:]}
                    content_blocks.append(current_section)
                elif line.startswith("## "):
                    current_section = {"type": "heading", "level": 2, "text": line[3:]}
                    content_blocks.append(current_section)
                elif line.startswith("- "):
                    if current_section and current_section["type"] == "list":
                        current_section["items"].append(line[2:])
                    else:
                        current_section = {"type": "list", "ordered": False, "items": [line[2:]]}
                        content_blocks.append(current_section)
                elif line.startswith("> "):
                    content_blocks.append({"type": "quote", "text": line[2:], "citation": "Generated Insight"})
                elif line.startswith("FEATURED_IMAGE_URL") and image_count == 0:
                    content_blocks.append({
                        "type": "image",
                        "url": featured_image,
                        "alt": f"Featured image for {topic}",
                        "caption": f"{topic} Overview"
                    })
                    image_count += 1
                elif line.startswith("CONTENT_IMAGE_URL") and image_count == 1:
                    content_blocks.append({
                        "type": "image",
                        "url": content_image,
                        "alt": f"Visual representation of {topic}",
                        "caption": f"Exploring {topic}"
                    })
                    image_count += 1
                else:
                    content_blocks.append({"type": "paragraph", "text": line})

            # Ensure exactly 2 images if not already present
            has_featured = any(b["type"] == "image" and b["caption"].startswith(f"{topic} Overview") for b in content_blocks)
            has_content = any(b["type"] == "image" and b["caption"].startswith(f"Exploring {topic}") for b in content_blocks)
            intro_end = 0
            for i, block in enumerate(content_blocks):
                if block["type"] == "heading" and i > 0:
                    intro_end = i
                    break

            if not has_featured:
                content_blocks.insert(0, {
                    "type": "image",
                    "url": featured_image,
                    "alt": f"Featured image for {topic}",
                    "caption": f"{topic} Overview"
                })
            if not has_content and intro_end > 0:
                content_blocks.insert(intro_end, {
                    "type": "image",
                    "url": content_image,
                    "alt": f"Visual representation of {topic}",
                    "caption": f"Exploring {topic}"
                })

            blog = {
                "title": f"Exploring {topic}: A Deep Dive",
                "content": content_blocks,
                "featuredImage": featured_image,
                "tags": [topic, "Insights", "Overview"],
                "summary": f"A comprehensive blog post exploring {topic}, its developments, impacts, and future possibilities."
            }
            return blog

def main():
    input_data = json.loads(sys.stdin.read())
    topic = input_data.get("topic", "")
    contents = input_data.get("contents", [])

    service = LlamaIndexService()
    documents = service.create_documents_from_scraped_content(contents)
    index = service.create_index(documents)
    blog = service.generate_blog_from_query(topic, index)

    print(json.dumps(blog))

if __name__ == "__main__":
    main()