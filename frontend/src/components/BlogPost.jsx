import React from "react";
import { Card, CardContent } from "@/components/ui/card";

function BlogPost({ blog }) {
  if (!blog) return null;

  // Function to parse content and handle different block types
  const renderContent = () => {
    // If content is a string (for backward compatibility)
    if (typeof blog.content === "string") {
      return blog.content.split("\n\n").map((paragraph, idx) => (
        <p key={idx} className="mb-4 text-gray-800 leading-relaxed">
          {paragraph}
        </p>
      ));
    }

    // If content is an array of structured blocks
    return blog.content.map((block, idx) => {
      switch (block.type) {
        case "heading": {
          const HeadingTag = `h${block.level}`;
          return (
            <HeadingTag
              key={idx}
              className={`font-bold text-gray-900 mb-4 mt-6 ${
                block.level === 1
                  ? "text-3xl"
                  : block.level === 2
                  ? "text-2xl"
                  : block.level === 3
                  ? "text-xl"
                  : "text-lg"
              }`}
            >
              {block.text}
            </HeadingTag>
          );
        }

        case "paragraph":
          return (
            <p key={idx} className="mb-4 text-gray-800 leading-relaxed">
              {block.text}
            </p>
          );

        case "image":
          return (
            <div key={idx} className="my-6">
              <img
                src={block.url}
                alt={block.alt || "Blog image"}
                className="rounded-lg w-full max-h-96 object-cover"
                onError={(e) => {
                  console.error(`Failed to load image: ${block.url}`);
                  e.target.onerror = null; 
                  e.target.src = 'https://placehold.co/600x400?text=Image+Not+Available';
                }}
              />
              {block.caption && (
                <p className="text-sm text-gray-500 mt-2 text-center italic">
                  {block.caption}
                </p>
              )}
            </div>
          );

        case "quote":
          return (
            <blockquote
              key={idx}
              className="border-l-4 border-gray-300 pl-4 italic my-6 text-gray-700"
            >
              {block.text}
              {block.citation && (
                <footer className="text-sm mt-2 text-gray-500">
                  — {block.citation}
                </footer>
              )}
            </blockquote>
          );

        case "list": {
          const ListTag = block.ordered ? "ol" : "ul";
          return (
            <ListTag
              key={idx}
              className={`mb-4 pl-6 ${
                block.ordered ? "list-decimal" : "list-disc"
              }`}
            >
              {block.items.map((item, itemIdx) => (
                <li key={itemIdx} className="mb-1 text-gray-800">
                  {item}
                </li>
              ))}
            </ListTag>
          );
        }

        default:
          return (
            <p key={idx} className="mb-4 text-gray-800">
              {block.text || JSON.stringify(block)}
            </p>
          );
      }
    });
  };

  return (
    <Card className="shadow-md bg-white">
      <CardContent className="p-0">
        {blog.featuredImage && (
          <div className="w-full h-64 md:h-80 relative overflow-hidden">
            <img
              src={blog.featuredImage}
              alt={blog.title || "Featured image"}
              className="rounded-lg w-full max-h-96 object-cover"
              onError={(e) => {
                console.error(`Failed to load featured image: ${blog.featuredImage}`);
                e.target.onerror = null;
                e.target.src = 'https://placehold.co/600x400?text=Image+Not+Available';
              }}
            />
          </div>
        )}

        <div className="p-6 md:p-8">
          <div className="mb-6">
            <h1 className="text-3xl md:text-4xl font-bold text-gray-900 mb-3">
              {blog.title}
            </h1>

            <div className="flex items-center text-sm text-gray-500 mb-4">
              {blog.date && (
                <span className="mr-4">
                  {new Date(blog.date).toLocaleDateString("en-US", {
                    year: "numeric",
                    month: "long",
                    day: "numeric",
                  })}
                </span>
              )}

              {blog.author && <span className="mr-4">By {blog.author}</span>}

              {blog.readingTime && <span>{blog.readingTime} min read</span>}
            </div>

            {blog.summary && (
              <div className="text-lg text-gray-700 font-medium border-l-4 border-gray-200 pl-4 italic">
                {blog.summary}
              </div>
            )}
          </div>

          <div className="prose max-w-none">{renderContent()}</div>

          {blog.tags && blog.tags.length > 0 && (
            <div className="mt-8 pt-6 border-t border-gray-200">
              <div className="flex flex-wrap gap-2">
                {blog.tags.map((tag, idx) => (
                  <span
                    key={idx}
                    className="bg-gray-100 text-gray-800 px-3 py-1 rounded-full text-sm"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export default BlogPost;
