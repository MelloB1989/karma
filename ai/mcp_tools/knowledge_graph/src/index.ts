import { Container, getContainer } from "@cloudflare/containers";
import { Hono } from "hono";
import { cors } from "hono/cors";

export class MyContainer extends Container {
  // Port the container listens on (default: 8080)
  defaultPort = 8080;
  // Time before container sleeps due to inactivity (default: 30s)
  sleepAfter = "2m";
  // Environment variables passed to the container
  envVars = {
    MESSAGE: "I was passed in via the container class!",
  };

  // Optional lifecycle hooks
  override onStart() {
    console.log("Container successfully started");
  }

  override onStop() {
    console.log("Container successfully shut down");
  }

  override onError(error: unknown) {
    console.log("Container error:", error);
  }
}

// Create Hono app with proper typing for Cloudflare Workers
const app = new Hono<{
  Bindings: { MY_CONTAINER: DurableObjectNamespace<MyContainer> };
}>();

// Disable CORS
app.use(
  "*",
  cors({
    origin: "*",
    allowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allowHeaders: ["*"],
  }),
);

app.post("/mcp", async (c) => {
  const container = getContainer(c.env.MY_CONTAINER);
  return await container.fetch(c.req.raw);
});

app.get("/mcp", async (c) => {
  const container = getContainer(c.env.MY_CONTAINER);
  return await container.fetch(c.req.raw);
});

export default app;
