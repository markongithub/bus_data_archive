package main

import (
	"encoding/gob"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"io/ioutil"
	"os"
	"regexp"
	// "strings"
	"time"
)

const timeFormat = "2006-01-02T15:04:05" // these are local

// This should be the output from GetBusesForRouteAll.jsp
type CleverPositionList struct {
	BusPositions []CleverPositionReport `xml:"bus"`
}

type CleverPositionReport struct {
	Vehicle       string  `xml:"id"`
	Route         string  `xml:"rt"`
	WhateverARIs  string  `xml:"ar"`
	WhateverDIs   string  `xml:"d"`
	DirectionDD   string  `xml:"dd"`
	DirectionDN   string  `xml:"dn"`
	Lat           float64 `xml:"lat"`
	Lon           float64 `xml:"lon"`
	PathID        string  `xml:"pid"`
	WhateverRunIs string  `xml:"run"`
	WhateverOPIs  string  `xml:"op"`
	BlockID       string  `xml:"bid"`
	HeadSign      string  `xml:"fs"`
}

type TripInstance struct {
	gorm.Model
	Vehicle      string `gorm:"index"`
	Route        string
	DirectionDD  string
	PathID       string
	Run          string
	HeadSign     string
	Op           string
	BlockID      string
	FirstSeen    time.Time
	LastSeen     time.Time
	TripIDGTFS   string
	BusPositions []BusPosition
}

type BusPosition struct {
	gorm.Model
	TripInstanceID uint `gorm:"index"`
	Lat            float64
	Lon            float64
	Deviation      float64
	Direction      string
	RetrievedAt    time.Time
}

type TripCache map[string]TripInstance

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func fileTime(filePath string) time.Time {
	// 6/buses2019-03-26T23:28:01.json
	r, _ := regexp.Compile("/buses(....-..-..T..:..:..)\\.xml")
	result := r.FindStringSubmatch(filePath)
	if result == nil {
		panic("could not parse time from inputFile")
	}
	t1, err := time.Parse(
		"2006-01-02T15:04:05", result[1])
	check(err)
	return t1

}

func matchTrips(t1 TripInstance, t2 TripInstance) bool {
	if (t1.DirectionDD == t2.DirectionDD) &&
		(t1.PathID == t2.PathID) &&
		(t1.Run == t2.Run) &&
		(t1.HeadSign == t2.HeadSign) &&
		(t1.Route == t2.Route) &&
		// this last one is crude. for a long ride, this means a 30 minute gap is
		// going to confuse us and make us think it's a new trip. GTFS would help
		// a lot. We could see if we expected the trip to still be going.
		// also this only works if you are adding reports in chronological order.
		(t2.LastSeen.Sub(t1.LastSeen).Minutes() < 30) {
		fmt.Printf("Matched with old trip %d last seen at %s", t1.ID, t1.LastSeen)
		return true
	}
	return false
}

func findExistingTrip(db *gorm.DB, cache TripCache, bpr CleverPositionReport, reportTime time.Time) (TripInstance, bool) {
	newTrip := tripFromReport(bpr, reportTime)
	cachedTrip, cacheHit := cache[bpr.Vehicle]
	if cacheHit && matchTrips(cachedTrip, newTrip) {
		return cachedTrip, true
	}
	oldTrip := TripInstance{}
	if (db.Where(TripInstance{Vehicle: bpr.Vehicle}).Order("last_seen desc").First(&oldTrip).RecordNotFound()) {
		return newTrip, false
	}
	// So we found the most recently seen trip by the same vehicle. Now we try to
	// figure out if this bus is on the same trip.
	if matchTrips(oldTrip, newTrip) {
		return oldTrip, true
	}
	return newTrip, false
}

func logPosition(db *gorm.DB, cache TripCache, bpr CleverPositionReport, reportTime time.Time) {
	var err error
	trip, tripFound := findExistingTrip(db, cache, bpr, reportTime)
	if !tripFound {
		fmt.Printf("We are seeing this trip for the first time.")
		db.NewRecord(trip)
		err = db.Create(&trip).Error
	} else {
		trip.LastSeen = reportTime
		err = db.Save(&trip).Error
	}
	cache[trip.Vehicle] = trip
	check(err)
	bp := BusPosition{
		RetrievedAt:    reportTime,
		Lat:            bpr.Lat,
		Lon:            bpr.Lon,
		Direction:      bpr.DirectionDN,
		TripInstanceID: trip.ID,
	}

	err = db.Model(&trip).Association("BusPositions").Append(bp).Error
	check(err)
}

func tripFromReport(bpr CleverPositionReport, reportTime time.Time) TripInstance {
	var route string
	if bpr.WhateverARIs != "" {
		route = bpr.WhateverARIs
	} else {
		route = bpr.Route
	}
	return TripInstance{
		Vehicle:     bpr.Vehicle,
		Route:       route,
		DirectionDD: bpr.DirectionDD,
		PathID:      bpr.PathID,
		Run:         bpr.WhateverRunIs,
		HeadSign:    bpr.HeadSign,
		FirstSeen:   reportTime,
		LastSeen:    reportTime,
		Op:          bpr.WhateverOPIs,
		BlockID:     bpr.BlockID,
	}

}

func parseFile(inputFile string) CleverPositionList {
	fmt.Printf("I will attempt to parse %s\n", inputFile)
	b, err := ioutil.ReadFile(inputFile)
	check(err)

	var m CleverPositionList
	err = xml.Unmarshal(b, &m)
	check(err)
	fmt.Printf("The file contains %d bus positions.\n", len(m.BusPositions))
	return m
}

func loadCache(filename string) TripCache {
	fmt.Printf("I will attempt to parse %s\n", filename)
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return make(TripCache)
	}
	var cache TripCache
	d := gob.NewDecoder(f)

	// Decoding the serialized data
	err = d.Decode(&cache)
	check(err)
	return cache
}

func writeCache(filename string, cache TripCache) {
	f, err := os.Create(filename)
	check(err)
	defer f.Close()

	e := gob.NewEncoder(f)
	err = e.Encode(cache)
	check(err)
}

func main() {
	inputFile := flag.String("input_file", "", "XML file with bus data")
	cacheFile := flag.String("cache_file", "", "serialized vehicle cache")
	flag.Parse()

	m := parseFile(*inputFile)
	reportTime := fileTime(*inputFile)

	// db, err := gorm.Open("postgres", os.Getenv("DB_CONFIG"))
	db, err := gorm.Open("sqlite3", "/tmp/gorm.db")
	check(err)
	defer db.Close()

	db.LogMode(true)

	db.AutoMigrate(&TripInstance{}, &BusPosition{})

	cache := loadCache(*cacheFile)

	fmt.Printf("The cache now has %d entries.\n", len(cache))
	for _, bp := range m.BusPositions {
		fmt.Printf("bus %s has head sign %s\n", bp.Vehicle, bp.HeadSign)
		if (bp.HeadSign == "Not in Service") || bp.HeadSign == "N/A" {
			fmt.Printf("We will skip that one.\n")
		} else {
			logPosition(db, cache, bp, reportTime)
		}
	}
	fmt.Printf("The cache now has %d entries.\n", len(cache))
	if *cacheFile != "" {
		writeCache(*cacheFile, cache)
	}
}
