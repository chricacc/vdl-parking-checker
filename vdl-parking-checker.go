package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"net/http"
)

type ParkingData struct {
	Parking []struct {
		Title  string `json:"title"`
		Occupied int    `json:"actuel"`
		Total  int    `json:"total"`
	} `json:"parking"`
}

type ParkingStatus struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	Total  int    `json:"total"`
	Occupied int    `json:"occupied"`
}


type Report struct {
	Timestamp string          `json:"timestamp"`
	Parkings  []ParkingStatus `json:"parkings"`
}

func fetchData(url string) (ParkingData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return ParkingData{}, fmt.Errorf("erreur HTTP : %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ParkingData{}, fmt.Errorf("statut HTTP invalide : %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ParkingData{}, fmt.Errorf("erreur de lecture : %v", err)
	}

	var data ParkingData
	if err := json.Unmarshal(body, &data); err != nil {
		return ParkingData{}, fmt.Errorf("erreur de parsing JSON : %v", err)
	}

	return data, nil
}

func determineStatus(occupied, total int) string {

	remaining := total - occupied
	switch {
	case remaining == 0:
		return "full"
	case remaining <= 20:
		return "almost-full"
	default:
		return "free"
	}
}

func loadPreviousReport(path string) (Report, error) {
	var r Report
	file, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(file, &r)
	return r, err
}

func saveReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func hasChanged(prev, curr Report) bool {
	if len(prev.Parkings) != len(curr.Parkings) {
		return true
	}
	statusMap := make(map[string]string)
	for _, p := range prev.Parkings {
		statusMap[p.Title] = p.Status
	}
	for _, p := range curr.Parkings {
		if statusMap[p.Title] != p.Status {
			return true
		}
	}
	return false
}

func sendNotification(webhookURL, message string) {
	payload := map[string]string{"content": message}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("Erreur webhook : %v\n", err)
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
		result[strings.TrimSpace(t)] = true
	}
	return result
}

func getMessageByStatus(p ParkingStatus) string {
	icon := ""
	switch{
	case p.Status == "free":
		icon = "✅"
	case p.Status == "almost-full":
		icon = "⚠️"
	case p.Status == "full":
		icon = "❌"
	}
	return fmt.Sprintf("%s Parking *%s* : %s (%d/%d places disponibles)", icon, p.Title, p.Status, p.Occupied, p.Total)
}

func main() {
	webhookURL := flag.String("webhook", "", "URL du webhook Discord ou Teams")
	titles := flag.String("titles", "", "Liste des parkings à surveiller, séparés par des virgules")
	dataURL := flag.String("dataUrl", "", "URL du fichier JSON à interroger")
	statusFile := flag.String("statusFile", "status.json", "Chemin du fichier de statut JSON")
	flag.Parse()

	if *webhookURL == "" || *titles == "" || *dataURL == "" {
		fmt.Println("Usage : ./bouillon_notifier -webhook <url> -titles <t1,t2> -dataUrl <url> [-statusFile path]")
		os.Exit(1)
	}

	titleSet := parseTitles(*titles)
	data, err := fetchData(*dataURL)
	if err != nil {
		fmt.Printf("Erreur récupération JSON : %v\n", err)
		os.Exit(1)
	}

	var currentReport Report
	currentReport.Timestamp = time.Now().UTC().Format(time.RFC3339)

	for _, p := range data.Parking {
		if titleSet[p.Title] {
			status := determineStatus(p.Occupied, p.Total)
			currentReport.Parkings = append(currentReport.Parkings, ParkingStatus{
				Title:  p.Title,
				Status: status,
				Total:  p.Total,
				Occupied: p.Occupied,
			})
		}
	}

	prevReport, err := loadPreviousReport(*statusFile)
	ignorePrev := false

	if err != nil {
		fmt.Println("Pas de rapport précédent (ou illisible), envoi forcé.")
		ignorePrev = true
	} else {
		prevDate := strings.Split(prevReport.Timestamp, "T")[0]
		currDate := strings.Split(currentReport.Timestamp, "T")[0]
		if prevDate != currDate {
			fmt.Println("Rapport précédent daté d’un autre jour, envoi forcé.")
			ignorePrev = true
		}
	}

	if !ignorePrev && !hasChanged(prevReport, currentReport) {
		fmt.Println("Aucun changement détecté, pas d’envoi de notification.")
		saveReport(*statusFile, currentReport)
		return
	}


	// Notification pour les changements
	for _, p := range currentReport.Parkings {
		msg := getMessageByStatus(p)
		sendNotification(*webhookURL, msg)
	}

	if err := saveReport(*statusFile, currentReport); err != nil {
		fmt.Printf("Erreur écriture rapport : %v\n", err)
	}
}
