package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jacaudi/nws/cmd/nws"
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

// RadarData represents the data for a radar.
type RadarData struct {
	Name              string // Name of the radar.
	VCP               string // Volume Coverage Pattern of the radar.
	Mode              string // Scanning mode of the radar.
	Status            string // Status of the radar.
	OperabilityStatus string // Operability Status of the radar.
	PowerSource       string // Power source of the radar.
	GenState          string // General state of the radar.
}

// checkEnvVars checks if the required environment variables are set.
// If any of the required variables are missing, it logs a fatal error.
func checkEnvVars() {
	var missingVars []string
	if !dryrun {
		if stationInput == "" {
			missingVars = append(missingVars, "STATION_IDS")
		}
		if apiToken == "" {
			missingVars = append(missingVars, "PUSHOVER_API_TOKEN")
		}
		if userKey == "" {
			missingVars = append(missingVars, "PUSHOVER_USER_KEY")
		}

		if len(missingVars) > 0 {
			log.Fatalf("The following environment variables are not set: %s", strings.Join(missingVars, ", "))
		}
	}
}

// sanitizeStationIDs splits a string of station IDs by space, comma, or semicolon
// and returns a slice of sanitized station IDs.
func sanitizeStationIDs(stationInput string) []string {
	// Define a regular expression to split by space, comma, or semicolon
	re := regexp.MustCompile(`[ ,;]+`)
	stationIDs := re.Split(stationInput, -1)
	for i := range stationIDs {
		stationIDs[i] = strings.TrimSpace(stationIDs[i])
	}
	return stationIDs
}

// getRadarResponse retrieves radar data for a given station ID.
// It returns a pointer to a RadarData struct and an error if any.
func getRadarResponse(stationID string) (*RadarData, error) {
	radarResponse, err := nws.RadarStation(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RADAR data for station %q: %w", stationID, err)
	}

	// Fetching radar VCP and determine mode
	radarVCPCode := radarResponse.RDA.Properties.VolumeCoveragePattern
	radarMode, err := radarMode(radarVCPCode) // Converting VCP to readable mode
	if err != nil {
		return nil, err
	}

	// Fetching radar VCP and determine mode
	genStateResponse := radarResponse.RDA.Properties.GeneratorState
	genStateStatement, err := genStateSimp(genStateResponse) // Converting generator state response to understandable text
	if err != nil {
		return nil, err
	}

	// Constructing the RadarData structure with both VCP and human-readable translation
	radarData := &RadarData{
		Name:              radarResponse.Name,
		VCP:               radarVCPCode,
		Mode:              radarMode,
		Status:            radarResponse.RDA.Properties.Status,
		OperabilityStatus: radarResponse.RDA.Properties.OperabilityStatus,
		PowerSource:       radarResponse.Performance.Properties.PowerSource,
		GenState:          genStateStatement,
	}

	return radarData, nil
}

// radarMode returns the radar mode based on the given VCP (Volume Coverage Pattern) code.
// It maps specific VCP codes to corresponding radar modes.
// If the VCP code is not recognized, it returns an error.
func radarMode(vcp string) (string, error) {
	var radarMode string

	switch vcp {
	case "R35":
		radarMode = "Clear Air"
	case "R212":
		radarMode = "Precipitation"
	case "R215":
		radarMode = "Precipitation"
	default:
		return "", fmt.Errorf("unknown mode for VCP %s -- please update code", vcp)
	}

	return radarMode, nil
}

// genStateSimp generates the simplified generator state based on the given input.
// It takes a genInput string as input and returns the corresponding genState string and an error (if any).
func genStateSimp(input string) (string, error) {
	replacements := map[string]string{
		"Switched to Auxiliary Power|Utility PWR Available|Generator On": "On",
		"Switched to Auxiliary Power|Generator On":                       "On",
		"Utility PWR Available|Generator On":                             "On",
		"Utility PWR Available":                                          "Off",
	}

	for pattern, replacement := range replacements {
		if input == pattern {
			return replacement, nil
		}
	}

	return "", errors.New("unknown input")
}

// compareRadarData compares the old and new radar data and returns whether there are any changes and the details of the changes.
// It takes two pointers to RadarData structs as input and returns a boolean indicating if there are any changes and a string containing the details of the changes.
func compareRadarData(oldData, newData *RadarData) (bool, string) {
	var changes []string

	if oldData.VCP != newData.VCP {
		if newData.VCP == "R35" {
			changes = append(changes, "The Radar is in Clear Air Mode -- No Precipitation Detected")
		} else if newData.VCP == "R215" {
			changes = append(changes, "The Radar is in Precipitation Mode -- Precipitation Detected")
		} else {
			changes = append(changes, fmt.Sprintf("Radar mode changed from %s to %s", oldData.VCP, newData.VCP))
		}
	}

	if oldData.Status != newData.Status {
		changes = append(changes, fmt.Sprintf("Radar status changed from %s to %s", oldData.Status, newData.Status))
	}

	if oldData.OperabilityStatus != newData.OperabilityStatus {
		changes = append(changes, fmt.Sprintf("Radar operability changed from %s to %s", oldData.OperabilityStatus, newData.OperabilityStatus))
	}

	if oldData.PowerSource != newData.PowerSource {
		changes = append(changes, fmt.Sprintf("Power source changed from %s to %s", oldData.PowerSource, newData.PowerSource))
	}

	if oldData.GenState != newData.GenState {
		changes = append(changes, fmt.Sprintf("Generator state changed from %s to %s", oldData.GenState, newData.GenState))
	}

	if len(changes) > 0 {
		return true, strings.Join(changes, "\n")
	}

	return false, ""
}

// sendPushoverNotification sends a Pushover notification with the specified title and message.
// It uses the Pushover service to send the notification.
// The function returns an error if the notification fails to send, otherwise it returns nil.
func sendPushoverNotification(title, message string) error {

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

	log.Println("Pushover notification sent successfully!")
	return nil
}

// fetchAndReportRadarData fetches radar data for a list of station IDs and reports any changes in the data.
// The fetched data is compared with the last stored data for each station ID, and if there are changes a
// push notification is sent using the sendPushoverNotification function.
// The radar data and its mode are stored in the radarDataMap in memory.
// Goroutines are used to perform the api call and data processing per station ID
func fetchAndReportRadarData(stationIDs []string, radarDataMap map[string]map[string]interface{}) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, stationID := range stationIDs {
		wg.Add(1)
		go func(stationID string) {
			defer wg.Done()
			log.Printf("Fetching radar data for station: %s\n", stationID)
			newRadarData, err := getRadarResponse(stationID)
			if err != nil {
				log.Printf("Error fetching radar data for station %s: %v\n", stationID, err)
				return
			}

			mode, err := radarMode(newRadarData.VCP)
			if err != nil {
				log.Printf("Error determining radar mode for station %s: %v\n", stationID, err)
				return
			}

			mu.Lock()
			if _, exists := radarDataMap[stationID]; !exists {
				radarDataMap[stationID] = make(map[string]interface{})
			}

			lastRadarData, exists := radarDataMap[stationID]["last"]
			if !exists || lastRadarData == nil {
				radarDataMap[stationID]["last"] = newRadarData
				mu.Unlock()
				initialMessage := fmt.Sprintf("%s %s - %s Mode", stationID, newRadarData.Name, mode)
				log.Printf("Initial radar data stored for station %s.", stationID)
				if dryrun {
					log.Printf("Debug Pushover -- Title: DRAS Startup - Msg: %s\n", initialMessage)
				} else {
					if err := sendPushoverNotification("DRAS Startup", initialMessage); err != nil {
						log.Fatalf("Error sending Pushover alert for station %s: %v\n", stationID, err)
					}
				}
				return
			}
			mu.Unlock()

			changed, changeMessage := compareRadarData(lastRadarData.(*RadarData), newRadarData)
			if changed {
				log.Printf("Radar data changed for station %s %s: %s\n", stationID, newRadarData.Name, changeMessage)
				if dryrun {
					log.Printf("Debug Pushover -- Title: %s - Msg: %s\n", stationID, changeMessage)
				} else {
					if err := sendPushoverNotification(fmt.Sprintf("%s Update", stationID), changeMessage); err != nil {
						log.Fatalf("Error sending Pushover alert for station %s: %v\n", stationID, err)
					}
				}
				mu.Lock()
				radarDataMap[stationID]["last"] = newRadarData
				mu.Unlock()
			} else {
				log.Printf("No changes in radar data for station %s\n", stationID)
			}
		}(stationID)
	}

	wg.Wait()
}

// The program checks environment variables, initializes variables, and starts the monitoring service.
// If the minuteInterval is not set, it defaults to 10 minutes.
// If dryrun is enabled, it uses test radar sites for monitoring.
// Otherwise, it sanitizes the station IDs provided by the user.
// It sets the UserAgent to the DRAS GitHub repository and fetches and reports radar data.
// It then starts a ticker to periodically update the radar data.
func main() {
	checkEnvVars()
	radarDataMap := make(map[string]map[string]interface{})
	var stationIDs []string

	if minuteInterval == 0 {
		minuteInterval = 10
	}

	log.Println("DRAS -- Start Monitoring Service")
	if dryrun {
		stationIDs = []string{"KATX", "KRAX"} // Test with Seattle, WA & Raleigh, NC Radar Sites
	} else {
		stationIDs = sanitizeStationIDs(stationInput)
	}
	log.Println("Set UserAgent to https://github.com/jacaudi/dras")
	config := nws.Config{}
	config.SetUserAgent("dras/1.0 (+https://github.com/jacaudi/dras)")
	fetchAndReportRadarData(stationIDs, radarDataMap)

	ticker := time.NewTicker(time.Duration(minuteInterval) * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		log.Println("DRAS -- Updating Radar Data")
		fetchAndReportRadarData(stationIDs, radarDataMap)
	}
}
