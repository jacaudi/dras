package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jacaudi/nwsgo"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/pushover"
)

var (
	stationInput      = os.Getenv("STATION_IDS")
	apiToken          = os.Getenv("PUSHOVER_API_TOKEN")
	userKey           = os.Getenv("PUSHOVER_USER_KEY")
	dryrun, _         = strconv.ParseBool(os.Getenv("DRYRUN"))
	minuteInterval, _ = strconv.ParseInt(os.Getenv("INTERVAL"), 10, 64)
)

// RadarData holds radar information, including both the raw VCP and its human-readable translation.
type RadarData struct {
	Name        string
	VCP         string
	Mode        string
	Status      string
	PowerSource string
	GenState    string
	FuelLevel   float64
}

// getRadarResponse fetches radar data for a given station and returns a processed RadarData structure.
func getRadarResponse(stationID string) (*RadarData, error) {
	radarResponse, err := nwsgo.RadarStation(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RADAR data for station %q: %w", stationID, err)
	}

	// Fetching radar properties
	radarVCPCode := radarResponse.RDA.Properties.VolumeCoveragePattern
	radarMode, err := radarMode(radarVCPCode) // Converting VCP to readable mode
	if err != nil {
		return nil, err
	}

	// Constructing the RadarData structure with both VCP and human-readable translation
	radarData := &RadarData{
		Name:        radarResponse.Name,
		VCP:         radarVCPCode,
		Mode:        radarMode,
		Status:      radarResponse.RDA.Properties.Mode,
		PowerSource: radarResponse.Performance.Properties.PowerSource,
		GenState:    radarResponse.RDA.Properties.GeneratorState,
		FuelLevel:   radarResponse.Performance.Properties.FuelLevel.Value,
	}

	return radarData, nil
}

// radarMode converts a VCP code into a human-readable radar mode description.
func radarMode(vcp string) (string, error) {
	var radarMode string

	switch vcp {
	case "R35":
		radarMode = "Clear Air"
	case "R215":
		radarMode = "Precipitation"
	default:
		return "", fmt.Errorf("unknown mode for VCP %s -- please update code", vcp)
	}

	return radarMode, nil
}

// compareRadarData compares two RadarData objects and returns a detailed message if they are different.
func compareRadarData(oldData, newData *RadarData) (bool, string) {
	if oldData.VCP != newData.VCP {
		if newData.VCP == "R35" {
			return false, "The Radar is in Precipitation Mode -- Precipitation Detected"
		} else if newData.VCP == "R215" {
			return false, "The Radar is in Clear Air Mode -- No Precipitation Detected"
		} else {
			return false, fmt.Sprintf("Radar mode changed from %s to %s", oldData.VCP, newData.VCP)
		}
	}

	if oldData.Status != newData.Status {
		return false, fmt.Sprintf("Radar status changed from %s to %s", oldData.Status, newData.Status)
	}

	if oldData.PowerSource != newData.PowerSource {
		return false, fmt.Sprintf("Power source changed from %s to %s", oldData.PowerSource, newData.PowerSource)
	}

	if oldData.GenState != newData.GenState {
		return false, fmt.Sprintf("Generator state changed from %s to %s", oldData.GenState, newData.GenState)
	}

	if oldData.FuelLevel != newData.FuelLevel {
		return false, fmt.Sprintf("Fuel level changed from %.2f to %.2f", oldData.FuelLevel, newData.FuelLevel)
	}

	return false, ""
}

func fetchAndReportRadarData(stationIDs []string, radarDataMap map[string]map[string]interface{}) {
	for _, stationID := range stationIDs {
		log.Printf("Fetching radar data for station: %s\n", stationID)
		newRadarData, err := getRadarResponse(stationID)
		if err != nil {
			log.Printf("Error fetching radar data for station %s: %v\n", stationID, err)
			continue
		}

		mode, err := radarMode(newRadarData.VCP)
		if err != nil {
			log.Printf("Error determining radar mode for station %s: %v\n", stationID, err)
			continue
		}

		if _, exists := radarDataMap[stationID]; !exists {
			radarDataMap[stationID] = make(map[string]interface{})
		}

		lastRadarData, exists := radarDataMap[stationID]["last"]
		if !exists || lastRadarData == nil {
			radarDataMap[stationID]["last"] = newRadarData
			initialMessage := fmt.Sprintf("%s %s - %s Mode", stationID, newRadarData.Name, mode)
			log.Printf("Initial radar data stored for station %s.", stationID)
			if dryrun {
				log.Printf("Debug Pushover Msg: %s\n", initialMessage)
			} else {
				if err := sendPushoverNotification("WSRif Startup", initialMessage); err != nil {
					log.Fatalf("Error sending Pushover alert for station %s: %v\n", stationID, err)
				}
			}
			continue
		}

		changed, changeMessage := compareRadarData(lastRadarData.(*RadarData), newRadarData)
		if changed {
			log.Printf("Radar data changed for station %s %s: %s\n", stationID, newRadarData.Name, changeMessage)
			if dryrun {
				log.Printf("Debug Pushover Msg: %s\n", changeMessage)
			} else {
				if err := sendPushoverNotification("WSRif Radar Update", changeMessage); err != nil {
					log.Fatalf("Error sending Pushover alert for station %s: %v\n", stationID, err)
				}
			}
			radarDataMap[stationID]["last"] = newRadarData
		} else {
			log.Printf("No changes in radar data for station %s\n", stationID)
		}
	}
}

// sendPushoverNotification sends a Pushover notification with the given title and message.
func sendPushoverNotification(title, message string) error {
	if apiToken == "" || userKey == "" {
		log.Fatalf("Pushover API token or user key is not set")
	}

	// Create a new Pushover service
	pushoverService := pushover.New(apiToken)

	// Add a recipient
	pushoverService.AddReceivers(userKey)

	// Create a new notification
	notification := notify.New()
	notification.UseServices(pushoverService)

	// Send the notification
	err := notification.Send(context.Background(), title, message)
	if err != nil {
		return err
	}

	log.Println("Notification sent successfully!")
	return nil
}

func main() {
	radarDataMap := make(map[string]map[string]interface{})
	var stationIDs []string

	if minuteInterval == 0 {
		minuteInterval = 10
	}

	log.Println("WSRif -- Start Radar Monitoring Service")
	if dryrun {
		stationIDs = []string{"KATX", "KRAX"} // Test with Seattle, WA & Raleigh, NC Radar Sites
	} else {
		stationIDs = strings.Split(stationInput, ",")
		if stationInput == "" {
			log.Fatalf("Error: STATION_IDS environment variable is not set or is empty")
		}
		for i := range stationIDs {
			stationIDs[i] = strings.TrimSpace(stationIDs[i])
		}
	}
	log.Println("Set UserAgent to https://github.com/jacaudi/wsrif")
	config := nwsgo.Config{}
	config.SetUserAgent("wsrif/1.0 (+https://github.com/jacaudi/wsrif)")
	fetchAndReportRadarData(stationIDs, radarDataMap)

	ticker := time.NewTicker(time.Duration(minuteInterval) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("WSRif -- Updating Radar Data")
		fetchAndReportRadarData(stationIDs, radarDataMap)
	}
}
