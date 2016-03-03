package main

import (
	"net/http"
	"log"
	"fmt"
	"crypto/tls"
	"encoding/json"
	"os"
	"net/url"
)

type Ingredient struct {
	ImageLink string
}

type Recipe struct {
	Id string
	Imagelink string
	Ingredients []Ingredient
}

type Collection struct {
	Items []Recipe
	Count int
}

func warmUp(w http.ResponseWriter, r *http.Request) {
	var token string
	var country string

	var server string
	server = os.Getenv("RESIZER_ENDPOINT")
	log.Printf("%s", server)
	if server == "" {
		FormatError(fmt.Errorf("Not host defined"), w)
		return
	}

	token = r.FormValue("token")
	log.Printf("Token %s", token)
	country = r.FormValue("country")
	log.Printf("Country %s", country)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	urlRequest := fmt.Sprintf("https://api-v2.hellofresh.com/recipes/dump?country=%s", country)
	req, _ := http.NewRequest("GET", urlRequest, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res, err := client.Do(req)

	defer res.Body.Close()
	defer r.Body.Close()

	if err != nil {
		FormatError(err, w)
		return
	}

	if res.StatusCode == 401 {
		FormatError(fmt.Errorf("Not allowed"), w)
		return
	}

	var collection Collection
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&collection)

	if err != nil {
		FormatError(err, w)
		return
	}

	log.Printf("Count: %d recipes", len(collection.Items))

	for i:= 0; i <= 99; i++ {
		log.Printf("Recipe: %s", collection.Items[i].Id)
		parseUrl, _ := url.Parse(collection.Items[i].Imagelink)
		timestamp := parseUrl.Query().Get("t")
		log.Printf("t: %s", timestamp)

		downloadImage(collection.Items[i].Id, server, timestamp)

		// ingredients too
		for _, ingredient := range collection.Items[i].Ingredients {
			ingredientId := ExtractIdFromUrl(ingredient.ImageLink)
			if ingredientId != "" {
				ingredientUrl := fmt.Sprintf("%s/image/%s.png", "200,200", ingredientId)
				downloadIngredients(ingredientUrl, server)
			}
		}
	}

	log.Printf("Warmup done for %s", country)
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, "Done!")
}

func downloadImage(id string, server string, timestamp string) {
	// we can then warmup those images
	for _, size := range config.Warmupsizes {
		imageUrl := fmt.Sprintf("%s/%d,%d/image/%s.jpg?t=%s", server, size.Width, size.Height, id, timestamp)
		res, _ := http.Get(imageUrl)

		if res.StatusCode == 200 {
			log.Printf("ImageUrl: %s", imageUrl)
		}

		defer res.Body.Close()
	}
}

func downloadIngredients (imageLink string, server string) {
	imageUrl := fmt.Sprintf("%s/%s", server, imageLink)

	res, _ := http.Get(imageUrl)

	if res.StatusCode == 200 {
		log.Printf("Ingredients saved: %s", imageUrl)
	}

	defer res.Body.Close()
}