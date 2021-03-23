package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"io/ioutil"
	"regexp"
	"time"
)

const timeFormat = "2006-01-02T15:04:05" // these are local

// This should be the output from https://developer.wmata.com/docs/services/54763629281d83086473f231/operations/5476362a281d830c946a3d68
type BusPositionList struct {
	BusPositions []BusPositionReport
}

type BusPositionReport struct {
	VehicleID     string
	TripID        string
	RouteID       string
	DirectionNum  int
	DirectionText string
	TripHeadSign  string
	TripStartTime string
	TripEndTime   string
	BlockNumber   string
	DateTime      string
	Lat           float64
	Lon           float64
	Deviation     float64
}

type TripInstance struct {
	gorm.Model
	VehicleID     string
	TripID        string `gorm:"unique_index:idx_trip_id_start_time"`
	RouteID       string
	DirectionNum  int
	DirectionText string
	TripHeadSign  string
	// need to think more before making this Time
	TripStartTime string `gorm:"unique_index:idx_trip_id_start_time"`
	TripEndTime   string
	BlockNumber   string
	BusPositions  []BusPosition
}

type BusPosition struct {
	gorm.Model
	RetrievedAt    time.Time
	ReportedAt     string
	TripInstanceID uint `gorm:"index"`
	Lat            float64
	Lon            float64
	Deviation      float64
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func fileTime(filePath string) time.Time {
	// 6/buses2019-03-26T23:28:01.json
	r, _ := regexp.Compile("/buses(....-..-..T..:..:..)\\.json")
	result := r.FindStringSubmatch(filePath)
	if result == nil {
		panic("could not parse time from filename")
	}
	t1, err := time.Parse(
		"2006-01-02T15:04:05", result[1])
	check(err)
	return t1

}

func tripFromReport(bpr BusPositionReport) TripInstance {
	return TripInstance{
		VehicleID:     bpr.VehicleID,
		TripID:        bpr.TripID,
		RouteID:       bpr.RouteID,
		DirectionNum:  bpr.DirectionNum,
		DirectionText: bpr.DirectionText,
		TripHeadSign:  bpr.TripHeadSign,
		TripStartTime: bpr.TripStartTime,
		TripEndTime:   bpr.TripEndTime,
		BlockNumber:   bpr.BlockNumber,
	}
}

func logPosition(db *gorm.DB, bpr BusPositionReport, reportTime time.Time) {
	var err error
	trip := tripFromReport(bpr)
	err = db.Where(TripInstance{TripID: trip.TripID, TripStartTime: trip.TripStartTime}).FirstOrCreate(&trip).Error
	check(err)
	bp := BusPosition{
		RetrievedAt: reportTime,
		ReportedAt:  bpr.DateTime,
		Lat:         bpr.Lat,
		Lon:         bpr.Lon,
		Deviation:   bpr.Deviation,
	}
	err = db.Model(&trip).Association("BusPositions").Append(bp).Error
	check(err)
}

func subtractDayFromStartTime(bpr BusPositionReport, startTime time.Time) BusPositionReport {
	output := bpr // we don't need a deep copy because bpr has no pointers
	output.TripStartTime = startTime.AddDate(0, 0, -1).Format(timeFormat)
	return output
}

func addDayToEndTime(bpr BusPositionReport, endTime time.Time) BusPositionReport {
	output := bpr // we don't need a deep copy because bpr has no pointers
	output.TripEndTime = endTime.AddDate(0, 0, 1).Format(timeFormat)
	return output
}

func addDayToBothTimes(bpr BusPositionReport, startTime time.Time, endTime time.Time, days int) BusPositionReport {
	output := bpr // we don't need a deep copy because bpr has no pointers
	output.TripStartTime = endTime.AddDate(0, 0, days).Format(timeFormat)
	output.TripEndTime = endTime.AddDate(0, 0, days).Format(timeFormat)
	return output
}

func fixBadTripData(bpr BusPositionReport, location *time.Location) (BusPositionReport, bool) {
	// The trick here is figuring out if the start time is bad or the end time is
	// bad, or both. If the start and end time are on either side of midnight,
	// they will corrupt the one that isn't "today". If the start and end time are
	// on the same day, they'll BOTH be corrupted to be "today".
	// A bad end time will be about 23 hours in the past, so let's
	// try to detect that first.
	reportTime, err := time.ParseInLocation(timeFormat, bpr.DateTime, location)
	check(err)
	startTime, err := time.ParseInLocation(timeFormat, bpr.TripStartTime, location)
	check(err)
	endTime, err := time.ParseInLocation(timeFormat, bpr.TripEndTime, location)
	check(err)
	if endTime.Before(startTime) {
		// Only one of them has been corrupted.
		fmt.Printf("Bad data at %s: start time %s is after end time %s\n", reportTime, startTime, endTime)
		// If the end time is more than 12 hours ago it's corrupt.
		if reportTime.Sub(endTime).Hours() > 12 {
			return addDayToEndTime(bpr, endTime), true
		}
		// A bad start time will typically be far into the future.
		if startTime.Sub(reportTime).Hours() > 12 {
			return subtractDayFromStartTime(bpr, startTime), true
		}
	}
	if startTime.Sub(reportTime).Hours() > 16 {

	}
	return BusPositionReport{}, false
}

func parseFile(filename string) BusPositionList {
	fmt.Printf("I will attempt to parse %s", filename)
	b, err := ioutil.ReadFile(filename)
	check(err)

	var m BusPositionList
	err = json.Unmarshal(b, &m)
	check(err)
	fmt.Printf("The file contains %d bus positions.\n", len(m.BusPositions))
	return m
}

func main() {
	filename := flag.String("input_file", "", "JSON file with bus data")
	onlyBadTripData := flag.Bool("bad_overnight_data_only", false, "ONLY process data with the overnight corruption bug")
	flag.Parse()

	m := parseFile(*filename)
	reportTime := fileTime(*filename)

	location, err := time.LoadLocation("US/Eastern")
	check(err)

	// TODO: This file name should be a flag.
	dbConfig, err := ioutil.ReadFile("./database_config.txt")
	check(err)

	db, err := gorm.Open("postgres", string(dbConfig[:]))
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	db.LogMode(true)

	db.AutoMigrate(&TripInstance{}, &BusPosition{})

	for _, bp := range m.BusPositions {
		fixed, isBad := fixBadTripData(bp, location)
		if isBad {
			logPosition(db, fixed, reportTime)
		} else {
			if !(*onlyBadTripData) {
				logPosition(db, bp, reportTime)
			} else {
				fmt.Println("Skipping since we are only backfilling bad trip data.")
			}

		}
	}
}
