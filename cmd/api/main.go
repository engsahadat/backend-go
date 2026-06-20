package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/your-org/ai-employee-platform/internal/database"
	authhttp "github.com/your-org/ai-employee-platform/internal/delivery/http"
)

// loadEnv reads key=value lines from a .env file and sets them as env variables.
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return // Ignore if file doesn't exist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			os.Setenv(key, val)
		}
	}
}

// generateSessionID generates a random 16-byte hex string.
func generateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "default"
	}
	return hex.EncodeToString(bytes)
}

// SaveFile saves base64 data to uploads/ directory and returns the public URL.
func SaveFile(base64Data string, mimeType string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", err
	}

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	filename := hex.EncodeToString(bytes)

	ext := ".bin"
	if strings.Contains(mimeType, "image/png") {
		ext = ".png"
	} else if strings.Contains(mimeType, "image/jpeg") || strings.Contains(mimeType, "image/jpg") {
		ext = ".jpg"
	} else if strings.Contains(mimeType, "image/webp") {
		ext = ".webp"
	} else if strings.Contains(mimeType, "image/gif") {
		ext = ".gif"
	} else if strings.Contains(mimeType, "video/mp4") {
		ext = ".mp4"
	} else if strings.Contains(mimeType, "video/webm") {
		ext = ".webm"
	} else if strings.Contains(mimeType, "audio/mpeg") || strings.Contains(mimeType, "audio/mp3") {
		ext = ".mp3"
	} else if strings.Contains(mimeType, "audio/wav") {
		ext = ".wav"
	} else if strings.Contains(mimeType, "application/pdf") {
		ext = ".pdf"
	} else if strings.Contains(mimeType, "text/plain") {
		ext = ".txt"
	}

	filePath := filepath.Join(uploadsDir, filename+ext)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	baseURL := os.Getenv("BACKEND_URL")
	if baseURL == "" {
		if os.Getenv("RENDER") == "true" {
			baseURL = "https://backend-go-9hto.onrender.com"
		} else {
			baseURL = "http://localhost:8080"
		}
	}

	return fmt.Sprintf("%s/uploads/%s%s", baseURL, filename, ext), nil
}

// corsMiddleware adds CORS headers to every response.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:3000":        true,
			"https://www.bdaiemployee.com": true,
			"https://bdaiemployee.com":     true,
		}

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// Fallback to wildcard for external API tests, or strict it
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var startTime = time.Now()

func main() {
	loadEnv()

	// Initialize database.
	if err := database.Init(); err != nil {
		log.Fatalf("❌ Database init failed: %v", err)
	}
	defer database.Close()

	mux := http.NewServeMux()

	// Register auth routes.
	authhttp.RegisterAuthRoutes(mux)

	// Root status page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		uptime := time.Since(startTime).Round(time.Second)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>AI Employee — API Server</title>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: 'Inter', sans-serif;
      background: #0a0a0f;
      color: #e2e8f0;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .card {
      background: linear-gradient(135deg, #12121e 0%%, #1a1a2e 100%%);
      border: 1px solid rgba(255,255,255,0.08);
      border-radius: 24px;
      padding: 56px 64px;
      text-align: center;
      max-width: 520px;
      width: 90%%;
      box-shadow: 0 32px 80px rgba(0,0,0,0.6), 0 0 0 1px rgba(99,102,241,0.15);
      position: relative;
      overflow: hidden;
    }
    .card::before {
      content: '';
      position: absolute;
      top: -50%%;
      left: -50%%;
      width: 200%%;
      height: 200%%;
      background: radial-gradient(circle at center, rgba(99,102,241,0.06) 0%%, transparent 60%%);
      pointer-events: none;
    }
    .pulse-ring {
      position: relative;
      width: 80px;
      height: 80px;
      margin: 0 auto 32px;
    }
    .pulse-ring::before {
      content: '';
      position: absolute;
      inset: -10px;
      border-radius: 50%%;
      border: 2px solid rgba(34,197,94,0.4);
      animation: ping 1.5s cubic-bezier(0,0,0.2,1) infinite;
    }
    @keyframes ping {
      75%%, 100%% { transform: scale(1.6); opacity: 0; }
    }
    .status-dot {
      width: 80px;
      height: 80px;
      border-radius: 50%%;
      background: linear-gradient(135deg, #22c55e, #16a34a);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 32px;
      box-shadow: 0 0 30px rgba(34,197,94,0.35);
    }
    h1 { font-size: 28px; font-weight: 700; color: #f1f5f9; margin-bottom: 8px; letter-spacing: -0.5px; }
    .subtitle { color: #64748b; font-size: 14px; margin-bottom: 36px; }
    .badge {
      display: inline-flex; align-items: center; gap: 6px;
      padding: 6px 14px; border-radius: 999px; font-size: 13px; font-weight: 600;
      background: rgba(34,197,94,0.12); color: #4ade80; border: 1px solid rgba(34,197,94,0.25);
      margin-bottom: 36px;
    }
    .badge-dot { width: 7px; height: 7px; border-radius: 50%%; background: #22c55e; animation: blink 1.2s ease-in-out infinite; }
    @keyframes blink { 0%%,100%% { opacity:1; } 50%% { opacity:0.3; } }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 32px; }
    .stat { background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.07); border-radius: 14px; padding: 16px; text-align: left; }
    .stat-label { font-size: 11px; color: #475569; text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 4px; }
    .stat-value { font-size: 15px; font-weight: 600; color: #cbd5e1; }
    .endpoints { background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.07); border-radius: 14px; padding: 16px 20px; text-align: left; }
    .endpoints-title { font-size: 11px; color: #475569; text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 12px; }
    .ep { display: flex; align-items: center; gap: 10px; padding: 6px 0; border-bottom: 1px solid rgba(255,255,255,0.05); font-size: 13px; }
    .ep:last-child { border-bottom: none; }
    .method { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 6px; min-width: 42px; text-align: center; }
    .get { background: rgba(34,197,94,0.15); color: #4ade80; }
    .post { background: rgba(99,102,241,0.15); color: #818cf8; }
    .path { color: #94a3b8; font-family: monospace; }
    .desc { color: #475569; margin-left: auto; font-size: 12px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="pulse-ring">
      <div class="status-dot">✓</div>
    </div>
    <h1>Server is Running</h1>
    <p class="subtitle">AI Employee Platform — Backend API</p>
    <div class="badge"><span class="badge-dot"></span> All systems operational</div>
    <div class="grid">
      <div class="stat"><div class="stat-label">Port</div><div class="stat-value">:8080</div></div>
      <div class="stat"><div class="stat-label">Uptime</div><div class="stat-value">%s</div></div>
      <div class="stat"><div class="stat-label">Database</div><div class="stat-value">SQLite (WAL)</div></div>
      <div class="stat"><div class="stat-label">Version</div><div class="stat-value">v1.1.0</div></div>
    </div>
    <div class="endpoints">
      <div class="endpoints-title">Available Endpoints</div>
      <div class="ep"><span class="method get">GET</span><span class="path">/health</span><span class="desc">Health check</span></div>
      <div class="ep"><span class="method post">POST</span><span class="path">/api/auth/register</span><span class="desc">Register</span></div>
      <div class="ep"><span class="method post">POST</span><span class="path">/api/auth/login</span><span class="desc">Login</span></div>
      <div class="ep"><span class="method post">POST</span><span class="path">/api/auth/google</span><span class="desc">Google OAuth</span></div>
      <div class="ep"><span class="method get">GET</span><span class="path">/api/auth/me</span><span class="desc">Current user</span></div>
      <div class="ep"><span class="method post">POST</span><span class="path">/chat</span><span class="desc">AI chat</span></div>
    </div>
  </div>
</body>
</html>`, uptime)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	// Serve static upload files
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// Fetch all chat sessions for the logged-in user
	mux.Handle("/api/chat/sessions", authhttp.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := authhttp.GetUserIDFromContext(r)

		rows, err := database.DB.Query(`
			SELECT session_id, MIN(created_at) as start_time, 
			       COALESCE((SELECT content FROM chat_history WHERE session_id = t.session_id AND user_id = t.user_id AND role = 'user' ORDER BY created_at ASC LIMIT 1), 'Untitled Chat') as title
			FROM chat_history t
			WHERE user_id = ?
			GROUP BY session_id
			ORDER BY start_time DESC`, userID)
		if err != nil {
			log.Printf("❌ [sessions] error fetching: %v", err)
			http.Error(w, `{"error":"failed to fetch sessions"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Session struct {
			SessionID string `json:"session_id"`
			StartTime string `json:"start_time"`
			Title     string `json:"title"`
		}

		sessions := []Session{}
		for rows.Next() {
			var s Session
			if err := rows.Scan(&s.SessionID, &s.StartTime, &s.Title); err != nil {
				log.Printf("❌ [sessions] error scanning: %v", err)
				http.Error(w, `{"error":"failed to parse sessions"}`, http.StatusInternalServerError)
				return
			}
			// Trim title if it's too long
			if len(s.Title) > 30 {
				s.Title = s.Title[:27] + "..."
			}
			sessions = append(sessions, s)
		}

		json.NewEncoder(w).Encode(sessions)
	})))

	// Fetch messages in a session OR delete a session
	mux.Handle("/api/chat/sessions/", authhttp.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := authhttp.GetUserIDFromContext(r)

		// Extract session_id from URL: /api/chat/sessions/{session_id}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.Error(w, `{"error":"session id required"}`, http.StatusBadRequest)
			return
		}
		sessionID := parts[4]

		if r.Method == http.MethodGet {
			rows, err := database.DB.Query(`
				SELECT role, content, metadata 
				FROM chat_history 
				WHERE user_id = ? AND session_id = ? 
				ORDER BY created_at ASC`, userID, sessionID)
			if err != nil {
				log.Printf("❌ [history] error fetching: %v", err)
				http.Error(w, `{"error":"failed to fetch history"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			type FileMetadata struct {
				Name string `json:"name"`
				Type string `json:"type"`
				URL  string `json:"url"`
			}
			type ChatMessage struct {
				Role    string        `json:"role"`
				Content string        `json:"content"`
				File    *FileMetadata `json:"file,omitempty"`
			}

			messages := []ChatMessage{}
			for rows.Next() {
				var msg ChatMessage
				var metaStr string
				if err := rows.Scan(&msg.Role, &msg.Content, &metaStr); err != nil {
					log.Printf("❌ [history] error scanning: %v", err)
					http.Error(w, `{"error":"failed to parse history"}`, http.StatusInternalServerError)
					return
				}

				if metaStr != "" && metaStr != "{}" {
					var metaObj struct {
						File *FileMetadata `json:"file"`
					}
					if err := json.Unmarshal([]byte(metaStr), &metaObj); err == nil && metaObj.File != nil {
						msg.File = metaObj.File
					}
				}
				messages = append(messages, msg)
			}

			json.NewEncoder(w).Encode(messages)
			return
		}

		if r.Method == http.MethodDelete {
			_, err := database.DB.Exec("DELETE FROM chat_history WHERE user_id = ? AND session_id = ?", userID, sessionID)
			if err != nil {
				log.Printf("❌ [history] error deleting: %v", err)
				http.Error(w, `{"error":"failed to delete session"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}

		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})))

	// Chat endpoint (with AuthMiddleware to load/save user chat history)
	mux.Handle("/chat", authhttp.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			Message   string `json:"message"`
			SessionID string `json:"session_id"` // Optional
			File      *struct {
				Data     string `json:"data"` // base64 encoded data
				MimeType string `json:"mime_type"`
				Name     string `json:"name"`
			} `json:"file"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		userID := authhttp.GetUserIDFromContext(r)
		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = generateSessionID()
		}

		log.Printf("[chat] user: %d, session: %s, message: %s (has file: %v)", userID, sessionID, req.Message, req.File != nil)

		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			log.Println("❌ [chat] error: GEMINI_API_KEY environment variable is not set")
			http.Error(w, `{"error":"AI service API key is not configured on the server."}`, http.StatusInternalServerError)
			return
		}

		// Handle file upload if any
		var fileURL string
		var err error
		if req.File != nil && req.File.Data != "" {
			fileURL, err = SaveFile(req.File.Data, req.File.MimeType)
			if err != nil {
				log.Printf("❌ [chat] error saving file: %v", err)
				http.Error(w, `{"error":"failed to save attached file"}`, http.StatusInternalServerError)
				return
			}
		}

		// Build parts list for Gemini
		parts := []map[string]interface{}{}
		if req.Message != "" {
			parts = append(parts, map[string]interface{}{
				"text": req.Message,
			})
		} else if req.File != nil {
			parts = append(parts, map[string]interface{}{
				"text": "Analyze this file in detail.",
			})
		}

		if req.File != nil && req.File.Data != "" {
			parts = append(parts, map[string]interface{}{
				"inlineData": map[string]interface{}{
					"mimeType": req.File.MimeType,
					"data":     req.File.Data,
				},
			})
		}

		geminiReq := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": parts,
				},
			},
			"systemInstruction": map[string]interface{}{
				"parts": []map[string]interface{}{
					{
						"text": "You are a helpful AI assistant. When a user asks you to create, generate, draw, or paint an image, you MUST return an inline image using markdown format. Use the following pattern: ![image](https://image.pollinations.ai/prompt/URL_ENCODED_PROMPT?width=1024&height=768&nologo=true). Ensure the prompt parameter in the URL is URL-encoded, highly descriptive, and doesn't contain unescaped spaces or special characters. Never say you cannot generate or display images; always generate the image using this Pollinations markdown syntax.",
					},
				},
			},
		}

		jsonData, err := json.Marshal(geminiReq)
		if err != nil {
			log.Printf("❌ [chat] error marshaling gemini request: %v", err)
			http.Error(w, `{"error":"failed to build AI request"}`, http.StatusInternalServerError)
			return
		}

		apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)
		
		var resp *http.Response
		var lastErr error
		maxRetries := 3
		backoff := 1 * time.Second

		for attempt := 1; attempt <= maxRetries; attempt++ {
			bodyReader := bytes.NewReader(jsonData)
			httpReq, err := http.NewRequest("POST", apiURL, bodyReader)
			if err != nil {
				log.Printf("❌ [chat] error creating outgoing request: %v", err)
				http.Error(w, `{"error":"failed to prepare AI request"}`, http.StatusInternalServerError)
				return
			}
			httpReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 60 * time.Second}
			resp, lastErr = client.Do(httpReq)
			
			if lastErr != nil {
				log.Printf("⚠️ [chat] Gemini API call attempt %d failed: %v", attempt, lastErr)
				if attempt < maxRetries {
					time.Sleep(backoff)
					backoff *= 2
					continue
				}
				break
			}

			// If success, we break
			if resp.StatusCode == http.StatusOK {
				break
			}
			
			// Retry on rate limit (429) or server errors (5xx)
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("⚠️ [chat] Gemini API returned status %d on attempt %d: %s", resp.StatusCode, attempt, string(bodyBytes))
				lastErr = fmt.Errorf("Gemini API returned status %d", resp.StatusCode)
				if attempt < maxRetries {
					time.Sleep(backoff)
					backoff *= 2
					continue
				}
				resp = nil
				break
			}
			
			// Do not retry on client errors (400, etc.)
			break
		}

		if lastErr != nil {
			log.Printf("❌ [chat] error calling Gemini API after %d attempts: %v", maxRetries, lastErr)
			if resp != nil {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("❌ [chat] Gemini API error response: %s", string(bodyBytes))
				http.Error(w, fmt.Sprintf(`{"error":"AI service error: status %d"}`, resp.StatusCode), resp.StatusCode)
			} else {
				http.Error(w, `{"error":"failed to connect to AI service after multiple attempts"}`, http.StatusInternalServerError)
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Printf("❌ [chat] Gemini API returned status %d: %s", resp.StatusCode, string(bodyBytes))
			http.Error(w, fmt.Sprintf(`{"error":"AI service error: status %d"}`, resp.StatusCode), resp.StatusCode)
			return
		}

		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
			log.Printf("❌ [chat] error parsing Gemini response: %v", err)
			http.Error(w, `{"error":"failed to read AI response"}`, http.StatusInternalServerError)
			return
		}

		var reply string
		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			reply = geminiResp.Candidates[0].Content.Parts[0].Text
		} else {
			reply = "I received an empty response from the AI."
		}

		// Save User Message to Database
		var metadataStr = "{}"
		if req.File != nil && fileURL != "" {
			metaMap := map[string]interface{}{
				"file": map[string]interface{}{
					"name": req.File.Name,
					"type": req.File.MimeType,
					"url":  fileURL,
				},
			}
			metaBytes, _ := json.Marshal(metaMap)
			metadataStr = string(metaBytes)
		}

		_, err = database.DB.Exec(`
			INSERT INTO chat_history (user_id, role, content, session_id, metadata) 
			VALUES (?, ?, ?, ?, ?)`,
			userID, "user", req.Message, sessionID, metadataStr)
		if err != nil {
			log.Printf("❌ [chat] error saving user message: %v", err)
		}

		// Save Assistant Reply to Database
		_, err = database.DB.Exec(`
			INSERT INTO chat_history (user_id, role, content, session_id) 
			VALUES (?, ?, ?, ?)`,
			userID, "assistant", reply, sessionID)
		if err != nil {
			log.Printf("❌ [chat] error saving assistant reply: %v", err)
		}

		json.NewEncoder(w).Encode(map[string]string{
			"reply":      reply,
			"session_id": sessionID,
		})
	})))

	log.Println("✅ Backend running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(mux)))
}
