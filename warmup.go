package main

import (
	"net/http"
	"log"
	"fmt"
	"crypto/tls"
	"encoding/json"
)

type Recipe struct {
	Id string
}

type Collection struct {
	Items []Recipe
	Count int
}

func warmUp(w http.ResponseWriter, r *http.Request) {
	var token string
	var country string

	token = r.FormValue("token")
	log.Printf("Token %s", token)
	country = r.FormValue("country")
	log.Printf("Country %s", country)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	url := fmt.Sprintf("https://api-v2.hellofresh.com/recipes/dump?country=%s", country)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res, err := client.Do(req)

	defer res.Body.Close()
	defer r.Body.Close()

	if err != nil {
		formatError(err, w)
		return
	}

	if res.StatusCode == 401 {
		formatError(fmt.Errorf("Not allowed"), w)
		return
	}

	var collection Collection
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&collection)

	if err != nil {
		formatError(err, w)
		return
	}

	log.Printf("Count: %d recipes", len(collection.Items))

	for i:= 0; i <= 99; i++ {
		log.Printf("Recipe: %s", collection.Items[i].Id)
		downloadImage(collection.Items[i].Id)
	}

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, "Done!")
}

func downloadImage(id string) {
	// we can then warmup those images
	for _, size := range config.Warmupsizes {
		imageUrl := fmt.Sprintf("http://127.0.0.1:8080/resize/%d,%d/image/%s.jpg", size.Width, size.Height, id)
		res, _ := http.Get(imageUrl)

		if res.StatusCode == 200 {
			log.Printf("ImageUrl: %s", imageUrl)
		}

		defer res.Body.Close()
	}
}
