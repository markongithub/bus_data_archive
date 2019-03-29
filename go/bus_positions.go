package main

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"io/ioutil"
	"os"
	"regexp"
	"time"
)

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

func logPosition(db *gorm.DB, bpr BusPositionReport, reportTime time.Time) {
	var trip TripInstance
	var err error
	trip = TripInstance{
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
func main() {
	filename := os.Args[1]
	b, err := ioutil.ReadFile(filename)
	check(err)

	var m BusPositionList
	err = json.Unmarshal(b, &m)
	check(err)
	fmt.Printf("The file contains %d bus positions.\n", len(m.BusPositions))
	reportTime := fileTime(filename)

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
		logPosition(db, bp, reportTime)
	}
}
