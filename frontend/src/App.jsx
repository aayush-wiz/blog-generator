import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Loader2 } from "lucide-react";
import BlogPost from "./components/BlogPost";

function App() {
  const [topic, setTopic] = useState("");
  const [loading, setLoading] = useState(false);
  const [blogData, setBlogData] = useState(null);
  const [error, setError] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!topic.trim()) return;

    setLoading(true);
    setError("");

    try {
      const response = await fetch("/api/generate-blog", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ topic }),
      });

      if (!response.ok) {
        throw new Error("Failed to generate blog post");
      }

      const data = await response.json();
      setBlogData(data);
    } catch (err) {
      setError(err.message || "Something went wrong");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 p-4 md:p-8">
      <div className="max-w-6xl mx-auto">
        <header className="mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            Blog Generator
          </h1>
          <p className="text-gray-600">
            Enter a topic to generate a comprehensive blog post with the latest
            information.
          </p>
        </header>

        <Card className="mb-8">
          <CardContent className="pt-6">
            <form
              onSubmit={handleSubmit}
              className="flex flex-col sm:flex-row gap-4"
            >
              <Input
                placeholder="Enter a blog topic (e.g., Climate Change Initiatives in 2025)"
                value={topic}
                onChange={(e) => setTopic(e.target.value)}
                className="flex-1"
              />
              <Button type="submit" disabled={loading || !topic.trim()}>
                {loading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Generating...
                  </>
                ) : (
                  "Generate Blog"
                )}
              </Button>
            </form>
            {error && <p className="mt-4 text-red-500">{error}</p>}
          </CardContent>
        </Card>

        {loading && (
          <div className="flex flex-col items-center justify-center py-12">
            <Loader2 className="h-12 w-12 animate-spin text-gray-400" />
            <p className="mt-4 text-lg text-gray-600">
              Researching and generating your blog post...
            </p>
            <p className="text-sm text-gray-500 mt-2">
              This may take a minute or two as we gather the latest information.
            </p>
          </div>
        )}

        {blogData && !loading && <BlogPost blog={blogData} />}
      </div>
    </div>
  );
}

export default App;
