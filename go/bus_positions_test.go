package main

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func getTimeZone() *time.Location {
	location, err := time.LoadLocation("US/Eastern")
	check(err)
	return location
}

func TestIsBadTripData(t *testing.T) {
	location := getTimeZone()
	m := parseFile("test_data/buses2019-04-27T03:55:01.json")
	assert.Equal(t, len(m.BusPositions), 2)
	// the first trip has sane data
	_, result := fixBadTripData(m.BusPositions[0], location)
	assert.Equal(t, result, false)
	// the second trip has bad data
	_, result = fixBadTripData(m.BusPositions[1], location)
	assert.Equal(t, result, true)
}

func TestFixBadTripDataBeforeMidnight(t *testing.T) {
	location := getTimeZone()
	m := parseFile("test_data/buses2019-04-27T03:55:01.json")
	badReport := m.BusPositions[1]
	fixed, isBad := fixBadTripData(badReport, location)
	assert.Equal(t, isBad, true)
	assert.Equal(t, fixed.TripEndTime, "2019-04-27T00:01:00")
	// the fixed report is not bad data anymore
	_, isBad = fixBadTripData(fixed, location)
	assert.Equal(t, isBad, false)
}

func TestFixBadTripDataAfterMidnight(t *testing.T) {
	location := getTimeZone()
	m := parseFile("test_data/buses2019-04-27T04:05:01.json")
	badReport := m.BusPositions[1]
	fixed, isBad := fixBadTripData(badReport, location)
	assert.Equal(t, isBad, true)
	assert.Equal(t, fixed.TripStartTime, "2019-04-26T23:24:00")
	// the fixed report is not bad data anymore
	_, isBad = fixBadTripData(fixed, location)
	assert.Equal(t, isBad, false)
}
