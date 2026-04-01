package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	pixelid "github.com/matthewblackburn/pixel-id/go"
)

func main() {
	machineID := uint16(1)
	if s := os.Getenv("MACHINE_ID"); s != "" {
		v, _ := strconv.ParseUint(s, 10, 16)
		machineID = uint16(v)
	}

	gen := pixelid.NewGenerator(pixelid.WithMachineID(machineID))

	mux := http.NewServeMux()

	// Generate a new ID and return it with avatar URLs.
	mux.HandleFunc("POST /api/id", func(w http.ResponseWriter, r *http.Request) {
		id, err := gen.Generate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		idStr := strconv.FormatInt(id, 10)
		ts, mid, seq := pixelid.ParseID(id)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":        idStr,
			"timestamp": ts.Format("2006-01-02T15:04:05.000Z07:00"),
			"machineId": mid,
			"sequence":  seq,
		})
	})

	// Render avatar as SVG or PNG.
	mux.HandleFunc("GET /api/avatar/{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")

		var idStr string
		var isPNG bool
		if strings.HasSuffix(path, ".svg") {
			idStr = strings.TrimSuffix(path, ".svg")
		} else if strings.HasSuffix(path, ".png") {
			idStr = strings.TrimSuffix(path, ".png")
			isPNG = true
		} else {
			http.Error(w, "use .svg or .png extension", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		// Parse optional query params.
		q := r.URL.Query()
		var opts []pixelid.AvatarOption

		if s := q.Get("size"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				opts = append(opts, pixelid.WithSize(v))
			}
		}
		if s := q.Get("grid"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				opts = append(opts, pixelid.WithGrid(v, v))
			}
		}
		if s := q.Get("colors"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v >= 1 {
				opts = append(opts, pixelid.WithColors(v))
			}
		}
		if q.Get("curves") == "true" || q.Get("curves") == "1" {
			opts = append(opts, pixelid.WithCurves(true))
		}

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		if isPNG {
			data, err := pixelid.RenderPNG(id, opts...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(data)
		} else {
			svg := pixelid.RenderSVG(id, opts...)
			w.Header().Set("Content-Type", "image/svg+xml")
			fmt.Fprint(w, svg)
		}
	})

	// Health check.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := ":8080"
	log.Printf("pixel-id API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
