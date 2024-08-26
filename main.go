package main

import (
	"crypto/sha1"
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed index.tmpl.html
var indexTemplateFS embed.FS

type indexTemplateParams struct {
	PokemonID   uint64
	PokemonName string
	Name        string
	Version     string
}

// env return environment value or default if not exists
func env(name string, def string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return def
}

func main() {
	addr := env("ADDR", "127.0.0.1")
	port := env("PORT", "8080")
	listen := fmt.Sprintf("%s:%s", addr, port)

	log.Printf("starting quelpoke app on http://%s", listen)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", index)
	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatal("failed to listen and serve:", err)
	}
}

// index renders the index template. It takes name in query parameters
func index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	name := r.URL.Query().Get("name")
	tmpl, err := template.New("").ParseFS(indexTemplateFS, "*")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("[ERR] failed to parse embed fs:", err)
		return
	}

	params := indexTemplateParams{
		PokemonID: pokemonID(name, 151),
		Name:      name,
		Version:   env("VERSION", "dev"),
	}
	params.PokemonName, err = pokemonName(params.PokemonID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("[ERR] failed to get pokemon name:", err)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "index.tmpl.html", params); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("[ERR] failed to execute index template:", err)
		return
	}

	log.Printf("generated page in %s with pokemon id: %d for name: %s", time.Since(start).String(), params.PokemonID, params.Name)
}

// pokemonID computes the sha1 sum of the name and return
// the modulo by m (m is the maximum pokemon id)
func pokemonID(name string, m uint64) uint64 {
	hasher := sha1.New()
	hasher.Write([]byte(name))
	return binary.BigEndian.Uint64(hasher.Sum(nil)) % m
}

func pokemonName(id uint64) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%d", id), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	type pokemon struct {
		Name string `json:"name"`
	}

	poke := new(pokemon)
	err = json.NewDecoder(resp.Body).Decode(poke)
	if err != nil {
		return "", err
	}

	return poke.Name, nil
}
