package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type voiceRequest struct {
	Sentences string `json:"sentences"`
}

type apiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type voiceResponseData struct {
	Op      string `json:"op"`
	Content string `json:"content"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/voice", voiceHandler)

	addr := ":" + envOrDefault("PORT", "8080")
	log.Printf("VoxCanvas backend demo listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func homeHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("VoxCanvas backend demo is running.\n"))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Service: "voxcanvas-backend",
		Status:  "ok",
	})
}

func voiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req voiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			Code: http.StatusBadRequest,
			Msg:  "invalid request body",
			Data: nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code: http.StatusOK,
		Msg:  "success",
		Data: voiceResponseData{
			Op:      "requirement",
			Content: req.Sentences,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
