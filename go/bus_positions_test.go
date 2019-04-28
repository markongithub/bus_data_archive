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
	result, err := isBadTripData(tripFromReport(m.BusPositions[0]), location)
	assert.NilError(t, err)
	assert.Equal(t, result, false)
  // the second trip has bad data
	result, err = isBadTripData(tripFromReport(m.BusPositions[1]), location)
	assert.NilError(t, err)
	assert.Equal(t, result, true)
}
