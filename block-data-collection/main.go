package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IshiniKiridena/block_data/datacollector"
)

func main() {
	//pass the ISO date and time as a string in the format of -> 2022-01-01T00:00:00Z
	//01st April 2023 to 02nd of April 2023

	//datacollector.CollectData("2023-04-01T00:00:00Z", "2023-04-02T00:00:00Z")
	//datacollector.GasDataCollector("2023-06-29T06:00:00Z", "2023-06-29T06:05:00Z")

	for {
		// To collect data every day
		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		yesterday := today.Add(-24 * time.Hour)

		todayString := today.Format("2006-01-02T15:04:05Z")
		yesterdayString := yesterday.Format("2006-01-02T15:04:05Z")

		done := make(chan bool)
		datacollector.GasDataCollector(yesterdayString, todayString, done)

		// Wait for data collection to be finished
		<-done

		// Remove csv files older than 2 months
		dataFolder := "tracified-scripts/block-data-collection"
		MonthsAgo := time.Now().AddDate(0, -2, 0)

		err := filepath.Walk(dataFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Check if the filepath is csv and older than a month.
			if !info.IsDir() && filepath.Ext(path) == ".csv" && info.ModTime().Before(MonthsAgo) {
				// Remove file
				err := os.Remove(path)
				if err != nil {
					return err
				}
				fmt.Println("Removed File: ", path)
			}

			return nil
		})

		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			fmt.Println("File removal completed successfully.")
		}
	}

}
