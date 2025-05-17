package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ParkingData struct {
	Parking []struct {
		Title  string `json:"title"`
		Actuel int    `json:"actuel"`
		Total  int    `json:"total"`
	} `json:"parking"`
}

func fetchAndCheck(dataURL string, webhookURL string, titlesToCheck map[string]bool) {
	resp, err := http.Get(dataURL)
	if err != nil {
		fmt.Printf("Erreur lors de la requête HTTP : %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Statut HTTP inattendu : %s\n", resp.Status)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Erreur de lecture de la réponse : %v\n", err)
		return
	}

	var data ParkingData
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Erreur lors du parsing JSON : %v\n", err)
		return
	}

	for _, p := range data.Parking {
		if titlesToCheck[p.Title] {
			if (p.Actuel == p.Total) {
				fmt.Printf("%s vide", p.Title)
				msg := fmt.Sprintf("❌ Le parking *%s* est entièrement vide (%d places disponibles).", p.Title, p.Total-p.Actuel)
				sendNotification(webhookURL, msg)
			} else if  ((p.Total - p.Actuel) < 20)  {
				
				fmt.Printf("%s presque vide, %d places restantes", p.Title, p.Total-p.Actuel)
				msg := fmt.Sprintf("⚠️ Le parking *%s* est presque vide (%d places disponibles).", p.Title, p.Total-p.Actuel)
				sendNotification(webhookURL, msg)
			}
		}
	}
}

func sendNotification(webhookURL string, message string) {
	payload := map[string]string{"content": message}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("Erreur lors de l’envoi du webhook : %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		fmt.Printf("Erreur webhook (HTTP %d)\n", resp.StatusCode)
	}
}

func parseTitles(titles string) map[string]bool {
	result := make(map[string]bool)
	for _, t := range strings.Split(titles, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			result[t] = true
		}
	}
	return result
}

func main() {
	webhookURL := flag.String("webhook", "", "URL du webhook Discord ou Teams")
	titles := flag.String("titles", "", "Liste des parkings à surveiller, séparés par des virgules (ex: Bouillon,Gëlle Fra)")
	dataURL := flag.String("dataUrl", "", "URL du fichier JSON à interroger")

	flag.Parse()

	if *webhookURL == "" || *titles == "" || *dataURL == "" {
		fmt.Println("Utilisation : ./bouillon_notifier -webhook <URL> -titles <Titre1,Titre2> -dataUrl <JSON URL>")
		os.Exit(1)
	}

	titleMap := parseTitles(*titles)
	fetchAndCheck(*dataURL, *webhookURL, titleMap)
}

