package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/parnurzeal/gorequest"
)

type ArtistInfo struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

type TrackInfo struct {
	Name   string     `json:"name"`
	Lyrics string     `json:"lyrics"`
	Artist ArtistInfo `json:"artist"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := mux.NewRouter()
	router.HandleFunc("/track/{region}", GetTopTrackInfo).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server started at port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func GetTopTrackInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	region := mux.Vars(r)["region"]
	lastFMAPIKey := os.Getenv("LASTFM_API_KEY")
	musixmatchAPIKey := os.Getenv("MUSIXMATCH_API_KEY")

	// insert top track region from Last.fm
	lastfmURL := fmt.Sprintf("hereputurlviaaccount", region, lastFMAPIKey)
	_, body, errs := gorequest.New().Get(lastfmURL).End()
	if errs != nil {
		http.Error(w, errs[0].Error(), http.StatusInternalServerError)
		return
	}

	// Track JSON response
	var tracksResp struct {
		TopTracks struct {
			Track []struct {
				Name   string                `json:"name"`
				Artist struct{ Name string } `json:"artist"`
			} `json:"track"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal([]byte(body), &tracksResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(tracksResp.TopTracks.Track) == 0 {
		http.Error(w, "No tracks found", http.StatusNotFound)
		return
	}

	// insert lyrics track from Musixmatch
	trackName := tracksResp.TopTracks.Track[0].Name
	artistName := tracksResp.TopTracks.Track[0].Artist.Name
	musixmatchURL := fmt.Sprintf("hereputurlviaaccount", trackName, artistName, musixmatchAPIKey)
	_, lyricsBody, lyricsErrs := gorequest.New().Get(musixmatchURL).End()
	if lyricsErrs != nil {
		http.Error(w, lyricsErrs[0].Error(), http.StatusInternalServerError)
		return
	}

	// make lyrics JSON response
	var lyricsResp struct {
		Message struct {
			Body struct {
				Lyrics struct {
					LyricsBody string `json:"lyrics_body"`
				} `json:"lyrics"`
			} `json:"body"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(lyricsBody), &lyricsResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// insert artist details from Last.fm
	artistInfoURL := fmt.Sprintf("hereputurlviaaccount", artistName, lastFMAPIKey)
	_, artistBody, artistErrs := gorequest.New().Get(artistInfoURL).End()
	if artistErrs != nil {
		http.Error(w, artistErrs[0].Error(), http.StatusInternalServerError)
		return
	}

	// artist details JSON response
	var artistResp struct {
		Artist struct {
			Name  string `json:"name"`
			Image []struct {
				URL string `json:"#text"`
			} `json:"image"`
		} `json:"artist"`
	}
	if err := json.Unmarshal([]byte(artistBody), &artistResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// make response
	response := TrackInfo{
		Name:   trackName,
		Lyrics: lyricsResp.Message.Body.Lyrics.LyricsBody,
		Artist: ArtistInfo{
			Name:     artistName,
			ImageURL: artistResp.Artist.Image[3].URL,
		},
	}

	json.NewEncoder(w).Encode(response)
}
