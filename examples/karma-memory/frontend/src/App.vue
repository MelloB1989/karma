<script setup>
import { ref, onMounted, watch, nextTick } from "vue";
import axios from "axios";
import {
    Send,
    Settings,
    Trash2,
    Cpu,
    Database,
    Zap,
    Brain,
    Activity,
    Clock,
} from "lucide-vue-next";

// State
const messages = ref([]);
const inputMessage = ref("");
const isLoading = ref(false);
const isStreaming = ref(false);
const sessionId = ref("");
const chatContainer = ref(null);

// Configuration State
const config = ref({
    retrievalMode: "auto", // 'auto' | 'conscious'
    cacheMode: "memory", // 'none' | 'memory' | 'redis'
});

// API Base URL (assuming backend runs on 8080)
const API_URL = "https://kmai.apps.mellob.in/api";

// Initialize Session
onMounted(() => {
    // Generate or retrieve session ID
    let storedSession = localStorage.getItem("karma_session_id");
    if (!storedSession) {
        storedSession = "sess_" + Math.random().toString(36).substr(2, 9);
        localStorage.setItem("karma_session_id", storedSession);
    }
    sessionId.value = storedSession;

    // Add initial greeting
    messages.value.push({
        role: "assistant",
        content: "Hello! I am Karma. How can I help you today?",
        timestamp: new Date(),
    });

    // Sync initial config
    updateConfig();
});

// Auto-scroll to bottom
const scrollToBottom = async () => {
    await nextTick();
    if (chatContainer.value) {
        chatContainer.value.scrollTop = chatContainer.value.scrollHeight;
    }
};

// Send Message
const sendMessage = async () => {
    if (!inputMessage.value.trim() || isLoading.value) return;

    const userMsg = inputMessage.value.trim();
    inputMessage.value = "";

    // Add user message
    messages.value.push({
        role: "user",
        content: userMsg,
        timestamp: new Date(),
    });
    scrollToBottom();

    isLoading.value = true;
    isStreaming.value = true;

    // Create placeholder for assistant message
    const assistantMsgIndex =
        messages.value.push({
            role: "assistant",
            content: "",
            timestamp: new Date(),
            recallLatency: null,
            totalLatency: null,
            isTyping: true,
        }) - 1;

    try {
        const response = await fetch(`${API_URL}/chat`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                session_id: sessionId.value,
                message: userMsg,
            }),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split("\n\n");
            buffer = lines.pop(); // Keep the last incomplete chunk

            for (const line of lines) {
                if (line.startsWith("data: ")) {
                    const dataStr = line.slice(6);
                    try {
                        const event = JSON.parse(dataStr);

                        if (event.type === "recall") {
                            messages.value[assistantMsgIndex].recallLatency =
                                event.recall_latency_ms;
                        } else if (event.type === "token") {
                            messages.value[assistantMsgIndex].isTyping = false; // First token received
                            messages.value[assistantMsgIndex].content +=
                                event.content || "";
                            scrollToBottom();
                        } else if (event.type === "done") {
                            messages.value[assistantMsgIndex].totalLatency =
                                event.total_latency_ms;
                        } else if (event.type === "error") {
                            messages.value[assistantMsgIndex].content +=
                                `\n[Error: ${event.message}]`;
                        }
                    } catch (e) {
                        console.error("Error parsing SSE event:", e);
                    }
                }
            }
        }
    } catch (error) {
        console.error("Chat error:", error);
        messages.value[assistantMsgIndex].content =
            "Error: Failed to connect to server.";
    } finally {
        isLoading.value = false;
        isStreaming.value = false;
        messages.value[assistantMsgIndex].isTyping = false;
        scrollToBottom();
    }
};

// Update Configuration
const updateConfig = async () => {
    try {
        await axios.post(`${API_URL}/config`, {
            session_id: sessionId.value,
            retrieval_mode: config.value.retrievalMode,
            cache_mode: config.value.cacheMode,
        });
    } catch (error) {
        console.error("Config update error:", error);
    }
};

// Watch for config changes
watch(() => config.value.retrievalMode, updateConfig);
watch(() => config.value.cacheMode, updateConfig);

// Reset Memory
const resetMemory = async () => {
    if (!confirm("Are you sure you want to clear conversation history?"))
        return;

    try {
        await axios.post(`${API_URL}/reset`, {
            session_id: sessionId.value,
        });
        messages.value = [
            {
                role: "assistant",
                content: "Memory cleared. Starting fresh conversation.",
                timestamp: new Date(),
            },
        ];
    } catch (error) {
        console.error("Reset error:", error);
    }
};

const formatTime = (date) => {
    return new Date(date).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
    });
};
</script>

<template>
    <div
        class="min-h-screen bg-gray-950 text-gray-100 flex flex-col font-sans selection:bg-primary-500/30"
    >
        <!-- Header -->
        <header
            class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-md sticky top-0 z-10"
        >
            <div
                class="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between"
            >
                <div class="flex items-center gap-3">
                    <img
                        src="/karma.png"
                        alt="Karma Logo"
                        class="w-8 h-8 object-contain"
                    />
                    <div>
                        <h1 class="font-bold text-xl tracking-tight">
                            Karma<span class="text-primary-500">Memory</span>
                        </h1>
                        <p class="text-xs text-gray-400">
                            Long-term Memory for AI Agents
                        </p>
                    </div>
                </div>
                <div
                    class="flex items-center gap-2 text-xs text-gray-500 bg-gray-900 px-3 py-1.5 rounded-full border border-gray-800"
                >
                    <div
                        class="w-2 h-2 rounded-full bg-green-500 animate-pulse"
                    ></div>
                    <span>System Online</span>
                    <span class="mx-1">|</span>
                    <span class="font-mono"
                        >{{ sessionId.substr(0, 8) }}...</span
                    >
                </div>
            </div>
        </header>

        <main
            class="flex-1 max-w-7xl w-full mx-auto p-4 flex gap-6 overflow-hidden h-[calc(100vh-4rem)]"
        >
            <!-- Sidebar / Settings -->
            <aside
                class="w-80 flex-shrink-0 flex flex-col gap-6 overflow-y-auto pb-20 hidden md:flex"
            >
                <!-- Retrieval Mode -->
                <div
                    class="bg-gray-900/50 rounded-xl p-5 border border-gray-800"
                >
                    <div class="flex items-center gap-2 mb-4 text-primary-400">
                        <Brain class="w-5 h-5" />
                        <h2 class="font-semibold">Retrieval Mode</h2>
                    </div>

                    <div class="space-y-3">
                        <label
                            class="flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-all duration-200"
                            :class="
                                config.retrievalMode === 'auto'
                                    ? 'bg-primary-900/20 border-primary-500/50'
                                    : 'bg-gray-900 border-gray-800 hover:border-gray-700'
                            "
                        >
                            <input
                                type="radio"
                                v-model="config.retrievalMode"
                                value="auto"
                                class="mt-1 text-primary-500 focus:ring-primary-500 bg-gray-800 border-gray-700"
                            />
                            <div>
                                <span class="block font-medium text-sm"
                                    >Auto Mode</span
                                >
                                <span class="block text-xs text-gray-400 mt-1"
                                    >Fast, category-based retrieval. Best for
                                    general context.</span
                                >
                            </div>
                        </label>

                        <label
                            class="flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-all duration-200"
                            :class="
                                config.retrievalMode === 'conscious'
                                    ? 'bg-primary-900/20 border-primary-500/50'
                                    : 'bg-gray-900 border-gray-800 hover:border-gray-700'
                            "
                        >
                            <input
                                type="radio"
                                v-model="config.retrievalMode"
                                value="conscious"
                                class="mt-1 text-primary-500 focus:ring-primary-500 bg-gray-800 border-gray-700"
                            />
                            <div>
                                <span class="block font-medium text-sm"
                                    >Conscious Mode</span
                                >
                                <span class="block text-xs text-gray-400 mt-1"
                                    >LLM-driven dynamic queries. Smarter,
                                    context-aware filtering.</span
                                >
                            </div>
                        </label>
                    </div>
                </div>

                <!-- Cache Mode -->
                <div
                    class="bg-gray-900/50 rounded-xl p-5 border border-gray-800"
                >
                    <div class="flex items-center gap-2 mb-4 text-primary-400">
                        <Zap class="w-5 h-5" />
                        <h2 class="font-semibold">Caching Strategy</h2>
                    </div>

                    <div class="space-y-3">
                        <label
                            class="flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all duration-200"
                            :class="
                                config.cacheMode === 'none'
                                    ? 'bg-primary-900/20 border-primary-500/50'
                                    : 'bg-gray-900 border-gray-800 hover:border-gray-700'
                            "
                        >
                            <input
                                type="radio"
                                v-model="config.cacheMode"
                                value="none"
                                class="text-primary-500 focus:ring-primary-500 bg-gray-800 border-gray-700"
                            />
                            <span class="text-sm font-medium">Disabled</span>
                        </label>

                        <label
                            class="flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all duration-200"
                            :class="
                                config.cacheMode === 'memory'
                                    ? 'bg-primary-900/20 border-primary-500/50'
                                    : 'bg-gray-900 border-gray-800 hover:border-gray-700'
                            "
                        >
                            <input
                                type="radio"
                                v-model="config.cacheMode"
                                value="memory"
                                class="text-primary-500 focus:ring-primary-500 bg-gray-800 border-gray-700"
                            />
                            <div class="flex-1">
                                <span class="block text-sm font-medium"
                                    >In-Memory</span
                                >
                                <span class="text-xs text-gray-500"
                                    >Local RAM cache</span
                                >
                            </div>
                            <Cpu class="w-4 h-4 text-gray-500" />
                        </label>

                        <label
                            class="flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all duration-200"
                            :class="
                                config.cacheMode === 'redis'
                                    ? 'bg-primary-900/20 border-primary-500/50'
                                    : 'bg-gray-900 border-gray-800 hover:border-gray-700'
                            "
                        >
                            <input
                                type="radio"
                                v-model="config.cacheMode"
                                value="redis"
                                class="text-primary-500 focus:ring-primary-500 bg-gray-800 border-gray-700"
                            />
                            <div class="flex-1">
                                <span class="block text-sm font-medium"
                                    >Redis</span
                                >
                                <span class="text-xs text-gray-500"
                                    >Distributed cache</span
                                >
                            </div>
                            <Database class="w-4 h-4 text-gray-500" />
                        </label>
                    </div>
                </div>

                <!-- Actions -->
                <button
                    @click="resetMemory"
                    class="flex items-center justify-center gap-2 w-full py-3 px-4 rounded-lg border border-red-900/30 text-red-400 hover:bg-red-900/20 transition-colors text-sm font-medium"
                >
                    <Trash2 class="w-4 h-4" />
                    Clear Memory History
                </button>
            </aside>

            <!-- Chat Area -->
            <section
                class="flex-1 flex flex-col bg-gray-900/30 rounded-2xl border border-gray-800 overflow-hidden relative"
            >
                <!-- Messages -->
                <div
                    ref="chatContainer"
                    class="flex-1 overflow-y-auto p-6 space-y-6 scroll-smooth"
                >
                    <div
                        v-for="(msg, idx) in messages"
                        :key="idx"
                        class="flex gap-4 max-w-3xl mx-auto"
                        :class="msg.role === 'user' ? 'flex-row-reverse' : ''"
                    >
                        <!-- Avatar -->
                        <div
                            class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0"
                            :class="
                                msg.role === 'user'
                                    ? 'bg-primary-600'
                                    : 'bg-gray-700'
                            "
                        >
                            <span
                                v-if="msg.role === 'user'"
                                class="text-xs font-bold"
                                >U</span
                            >
                            <img
                                v-else
                                src="/karma.png"
                                class="w-5 h-5 object-contain"
                            />
                        </div>

                        <!-- Bubble & Metrics -->
                        <div class="flex flex-col gap-1 max-w-[80%]">
                            <!-- Message Bubble -->
                            <div
                                class="px-4 py-3 rounded-2xl text-sm leading-relaxed shadow-sm relative group"
                                :class="[
                                    msg.role === 'user'
                                        ? 'bg-primary-600 text-white rounded-tr-none'
                                        : msg.role === 'system'
                                          ? 'bg-red-900/20 text-red-300 border border-red-900/50'
                                          : 'bg-gray-800 text-gray-200 rounded-tl-none border border-gray-700',
                                ]"
                            >
                                <!-- Typing Indicator -->
                                <div
                                    v-if="msg.isTyping"
                                    class="flex items-center gap-1 h-5"
                                >
                                    <span
                                        class="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce"
                                        style="animation-delay: 0ms"
                                    ></span>
                                    <span
                                        class="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce"
                                        style="animation-delay: 150ms"
                                    ></span>
                                    <span
                                        class="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce"
                                        style="animation-delay: 300ms"
                                    ></span>
                                </div>

                                <!-- Content -->
                                <div v-else class="whitespace-pre-wrap">
                                    {{ msg.content }}
                                </div>
                            </div>

                            <!-- Metadata Row -->
                            <div
                                class="flex items-center gap-3 text-[10px] text-gray-500 px-1"
                                :class="
                                    msg.role === 'user'
                                        ? 'justify-end'
                                        : 'justify-start'
                                "
                            >
                                <span>{{ formatTime(msg.timestamp) }}</span>

                                <!-- Metrics (Assistant Only) -->
                                <template
                                    v-if="
                                        msg.role === 'assistant' &&
                                        (msg.recallLatency || msg.totalLatency)
                                    "
                                >
                                    <span
                                        class="w-1 h-1 rounded-full bg-gray-700"
                                    ></span>
                                    <div
                                        class="flex items-center gap-1 text-primary-400/80"
                                        title="Recall Latency"
                                    >
                                        <Brain class="w-3 h-3" />
                                        <span>{{
                                            msg.recallLatency
                                                ? msg.recallLatency + "ms"
                                                : "..."
                                        }}</span>
                                    </div>
                                    <span
                                        class="w-1 h-1 rounded-full bg-gray-700"
                                    ></span>
                                    <div
                                        class="flex items-center gap-1 text-green-400/80"
                                        title="Total Latency"
                                    >
                                        <Clock class="w-3 h-3" />
                                        <span>{{
                                            msg.totalLatency
                                                ? (
                                                      msg.totalLatency / 1000
                                                  ).toFixed(2) + "s"
                                                : "..."
                                        }}</span>
                                    </div>
                                </template>
                            </div>
                        </div>
                    </div>

                    <!-- Loading Animation (Initial Wait) -->
                    <div
                        v-if="
                            isLoading &&
                            messages[messages.length - 1]?.role === 'user'
                        "
                        class="flex gap-4 max-w-3xl mx-auto"
                    >
                        <div
                            class="w-8 h-8 rounded-full bg-gray-700 flex items-center justify-center flex-shrink-0"
                        >
                            <img
                                src="/karma.png"
                                class="w-5 h-5 object-contain animate-pulse"
                            />
                        </div>
                        <div
                            class="bg-gray-800 px-4 py-3 rounded-2xl rounded-tl-none border border-gray-700 flex items-center gap-2"
                        >
                            <Activity
                                class="w-4 h-4 text-primary-500 animate-spin"
                            />
                            <span class="text-xs text-gray-400"
                                >Recalling memories...</span
                            >
                        </div>
                    </div>
                </div>

                <!-- Input Area -->
                <div
                    class="p-4 bg-gray-900/80 backdrop-blur border-t border-gray-800"
                >
                    <div class="max-w-3xl mx-auto relative">
                        <input
                            v-model="inputMessage"
                            @keydown.enter="sendMessage"
                            type="text"
                            placeholder="Type a message to Karma..."
                            class="w-full bg-gray-800 text-gray-100 rounded-xl pl-4 pr-12 py-3.5 border border-gray-700 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 outline-none transition-all placeholder:text-gray-500 shadow-lg"
                            :disabled="isLoading"
                        />
                        <button
                            @click="sendMessage"
                            :disabled="!inputMessage.trim() || isLoading"
                            class="absolute right-2 top-1/2 -translate-y-1/2 p-2 rounded-lg text-primary-400 hover:bg-primary-900/30 hover:text-primary-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            <Send class="w-5 h-5" />
                        </button>
                    </div>
                    <p class="text-center text-xs text-gray-600 mt-3">
                        Karma Memory stores context from this conversation to
                        improve future responses.
                    </p>
                </div>
            </section>
        </main>

        <!-- Footer -->
        <footer
            class="py-4 text-center text-xs text-gray-600 border-t border-gray-900 bg-gray-950"
        >
            <p>
                Made by
                <span class="text-primary-500 font-medium">MelloB</span> with
                love ❤️
            </p>
        </footer>
    </div>
</template>

<style>
/* Custom radio button styling */
input[type="radio"] {
    appearance: none;
    background-color: transparent;
    margin: 0;
    font: inherit;
    color: currentColor;
    width: 1.15em;
    height: 1.15em;
    border: 0.15em solid currentColor;
    border-radius: 50%;
    display: grid;
    place-content: center;
}

input[type="radio"]::before {
    content: "";
    width: 0.65em;
    height: 0.65em;
    border-radius: 50%;
    transform: scale(0);
    transition: 120ms transform ease-in-out;
    box-shadow: inset 1em 1em currentColor;
}

input[type="radio"]:checked::before {
    transform: scale(1);
}
</style>
