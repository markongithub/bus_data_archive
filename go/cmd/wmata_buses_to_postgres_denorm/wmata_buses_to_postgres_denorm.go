package main

import (
	"flag"
	"fmt"
	"github.com/markongithub/bus_data_archive/pkg/bus_positions"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io/ioutil"
	"time"

	"encoding/json"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// This should be the output from https://developer.wmata.com/docs/services/54763629281d83086473f231/operations/5476362a281d830c946a3d68
type BusPositionList struct {
	BusPositions []BusPositionReport
}

type BusPositionReport struct {
	VehicleID     string
	TripID        string
	RouteID       string
	DirectionNum  json.Number
	DirectionText string
	TripHeadSign  string
	TripStartTime string
	TripEndTime   string
	BlockNumber   string
	DateTime      string
	Lat           json.Number
	Lon           json.Number
	Deviation     json.Number
}

type BusPositionReportSQLDenorm struct {
	gorm.Model
	BusPositionReport
	RetrievedAt time.Time
}

func ParseFile(filename string) BusPositionList {
	fmt.Printf("I will attempt to parse %s", filename)
	b, err := ioutil.ReadFile(filename)
	check(err)

	var m BusPositionList
	err = json.Unmarshal(b, &m)
	check(err)
	fmt.Printf("The file contains %d bus positions.\n", len(m.BusPositions))
	return m
}

func CheckInvariant(bpl BusPositionList) {
	seenVehicle := make(map[string]bool)
	for _, bpr := range bpl.BusPositions {
		if seenVehicle[bpr.VehicleID] {
			panic(fmt.Errorf("We saw vehicle %v twice in the same file.", bpr.VehicleID))
		} else {
			seenVehicle[bpr.VehicleID] = true
		}
	}
}

func ConvertToFlatRecord(b BusPositionReport, retrievedAt time.Time) BusPositionReportSQLDenorm {
	return BusPositionReportSQLDenorm{
		BusPositionReport: b,
		RetrievedAt:       retrievedAt,
	}
}

func logPosition(db *gorm.DB, bpr BusPositionReport, reportTime time.Time) {
	var err error
	record := ConvertToFlatRecord(bpr, reportTime)
	result := db.Create(&record)
}

func main() {
	filename := flag.String("input_file", "", "JSON file with bus data")
	flag.Parse()

	m := ParseFile(*filename)
	CheckInvariant(m)

	db, err := gorm.Open(postgres.Open(""), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	db.LogMode(true)

	db.AutoMigrate(&BusPositionReportSQLDenorm{})

	reportTime := bus_positions.FileTime(*filename)

	for _, bp := range m.BusPositions {
		logPosition(db, bp, reportTime)
	}
}
