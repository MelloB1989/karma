# Karma Memory UI Example

This example project demonstrates the capabilities of the **Karma Memory** system through a modern, dark-themed chat interface. It allows you to interact with an AI agent that remembers context across conversations and lets you dynamically switch between different memory retrieval and caching strategies.

## Features

- **Chat Interface**: Real-time chat with memory persistence.
- **Dynamic Configuration**:
  - **Retrieval Modes**: Switch between `Auto` (fast, category-based) and `Conscious` (smart, LLM-driven) retrieval.
  - **Caching Strategies**: Toggle between `Disabled`, `In-Memory`, and `Redis` caching.
- **Session Management**: Unique sessions per browser to isolate memory contexts.
- **Rate Limiting**: Basic IP-based rate limiting on the backend.

## Prerequisites

- **Go** (1.23+)
- **Node.js** (18+) & **npm**
- **OpenAI API Key** (for the LLM and Embeddings)
- **Redis** (Optional, for Redis caching mode)
- **Vector Database Credentials** (Pinecone or Upstash, configured via environment variables)

## Setup & Running

### 1. Backend (Go)

The backend handles the chat logic, memory management, and API endpoints.

1. Navigate to the backend directory:
   ```bash
   cd backend
   ```

2. Set your environment variables:
   ```bash
   export OPENAI_KEY="your-openai-key"
   
   # If using Pinecone
   export PINECONE_API_KEY="your-pinecone-key"
   export PINECONE_INDEX_HOST="your-index-host"
   
   # If using Upstash
   export UPSTASH_VECTOR_REST_URL="your-upstash-url"
   export UPSTASH_VECTOR_REST_TOKEN="your-upstash-token"

   # Optional: For Redis Caching
   export REDIS_URL="redis://localhost:6379"
   ```

3. Run the server:
   ```bash
   go run main.go
   ```
   The backend will start on `http://localhost:8080`.

### 2. Frontend (Vue.js + Tailwind)

The frontend provides the user interface.

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Start the development server:
   ```bash
   npm run dev
   ```
   The frontend will typically start on `http://localhost:5173`.

## Usage

1. Open your browser and navigate to the frontend URL (e.g., `http://localhost:5173`).
2. Start chatting with Karma!
   - Tell it your name or a fact about yourself.
   - Ask it to recall that information later.
3. Use the sidebar to experiment with different settings:
   - **Retrieval Mode**: See how "Conscious" mode handles complex queries compared to "Auto".
   - **Caching**: Enable caching to see if response times improve for repeated queries.
4. Use the "Clear Memory History" button to wipe the current session's memory.

## Architecture

- **Backend**: A simple Go HTTP server using `net/http`. It maintains stateful `KarmaMemory` instances mapped to session IDs.
- **Frontend**: A Vue 3 application using Vite and Tailwind CSS. It communicates with the backend via REST API.

## License

MIT