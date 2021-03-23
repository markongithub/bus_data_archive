package bus_positions

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
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

// These next two are actual bad data I got from WMATA. (The previous ones
// might also have been, I forget.)
// This test fails, as it should, because the code is still not correcting this
// case. Fix the bug or delete this test.
func TestBothCorruptBeforeMidnight(t *testing.T) {
	location := getTimeZone()
	m := parseFile("test_data/buses2019-09-19T03:59:02.json")
	badReport := m.BusPositions[0]
	fixed, isBad := fixBadTripData(badReport, location)
	assert.Equal(t, isBad, true)
	// 2019-09-18T00:01:00","TripEndTime":"2019-09-18T00:23:00"
	assert.Equal(t, fixed.TripStartTime, "2019-09-19T00:01:00")
	assert.Equal(t, fixed.TripEndTime, "2019-09-19T00:23:00")
	// the fixed report is not bad data anymore
	_, isBad = fixBadTripData(fixed, location)
	assert.Equal(t, isBad, false)
}

func TestBothCorruptAfterMidnight(t *testing.T) {
	location := getTimeZone()
	m := parseFile("test_data/buses2019-09-22T04:02:01.json")
	badReport := m.BusPositions[0]
	fixed, isBad := fixBadTripData(badReport, location)
	assert.Equal(t, isBad, true)
	// "TripStartTime":"2019-09-22T23:13:00","TripEndTime":"2019-09-22T23:55:00"
	assert.Equal(t, fixed.TripStartTime, "2019-09-21T23:13:00")
	assert.Equal(t, fixed.TripEndTime, "2019-09-21T23:55:00")
	// the fixed report is not bad data anymore
	_, isBad = fixBadTripData(fixed, location)
	assert.Equal(t, isBad, false)
}
