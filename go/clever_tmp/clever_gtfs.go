package main

import "github.com/artonge/go-gtfs"
import "fmt"
import "time"

func dateInRange(calendar gtfs.Calendar, dateYYYYMMDD string) bool {
	return (calendar.Start <= dateYYYYMMDD &&
		calendar.End >= dateYYYYMMDD)
}

func validForWeekday(calendar gtfs.Calendar, date time.Time) int {
	switch date.Weekday() {
	case 0:
		return calendar.Sunday
	case 1:
		return calendar.Monday
	case 2:
		return calendar.Tuesday
	case 3:
		return calendar.Wednesday
	case 4:
		return calendar.Thursday
	case 5:
		return calendar.Friday
	case 6:
		return calendar.Saturday
	default:
		panic("unexpected weekday integer value")
	}
}

func serviceIDsByDate(feed ReducedFeed, dateYYYYMMDD string) []string {
	date, err := time.Parse("20060102", dateYYYYMMDD)
	check(err)
	fmt.Printf("The date parsed as %s\n", date)
	validCalendars := make(map[string]bool)
	for _, calendar := range feed.Calendars {
		if dateInRange(calendar, dateYYYYMMDD) {
			if validForWeekday(calendar, date) != 0 {
				validCalendars[calendar.ServiceID] = true
			}
		}
	}
	for _, calendarDate := range feed.CalendarDates {
		if calendarDate.Date == dateYYYYMMDD {
			switch calendarDate.ExceptionType {
			case 1:
				validCalendars[calendarDate.ServiceID] = true
			case 2:
				delete(validCalendars, calendarDate.ServiceID)
			default:
				panic(fmt.Sprintf("Unexpected exception type: %s", calendarDate.ExceptionType))
			}
		}
	}
	output := make([]string, len(validCalendars))
	i := 0
	for k := range validCalendars {
		output[i] = k
		i++
	}

	return output
}

type StartEndTime = struct {
	StartTime string
	EndTime   string
}

func startEndTimesByTripID(stopTimes []gtfs.StopTime) map[string]StartEndTime {
	output := make(map[string]StartEndTime)
	for _, stopTime := range stopTimes {
		if curValue, present := output[stopTime.TripID]; present {
			if stopTime.Departure < curValue.StartTime {
				curValue.StartTime = stopTime.Departure
			}
			if stopTime.Departure > curValue.EndTime {
				curValue.EndTime = stopTime.Departure
			}
		} else {
			output[stopTime.TripID] = StartEndTime{stopTime.Departure, stopTime.Departure}
		}
	}
	return output
}

type ServiceIDAndHeadsign = struct {
	ServiceID string
	Headsign  string
}

func tripsByServiceIDAndHeadsign(trips []gtfs.Trip) map[ServiceIDAndHeadsign][]gtfs.Trip {
	output := make(map[ServiceIDAndHeadsign][]gtfs.Trip)
	for _, trip := range trips {
		index := ServiceIDAndHeadsign{trip.ServiceID, trip.Headsign}
		output[index] = append(output[index], trip)
	}
	return output
}

type ReducedFeed = struct {
	Calendars                   []gtfs.Calendar
	CalendarDates               []gtfs.CalendarDate
	TripsByServiceIDAndHeadsign map[ServiceIDAndHeadsign][]gtfs.Trip
	StartEndTimesByTripID       map[string]StartEndTime
}

func loadReducedFeed(gtfsPath string) ReducedFeed {
	feed, err := gtfs.Load(gtfsPath, nil)
	check(err)
	return ReducedFeed{
		feed.Calendars,
		feed.CalendarDates,
		tripsByServiceIDAndHeadsign(feed.Trips),
		startEndTimesByTripID(feed.StopsTimes),
	}
}

func guessTrip(feed ReducedFeed, headsign string, firstSeen time.Time, lastSeen time.Time) {

	return

}

func mainOrNotIGuess() {
	feed := loadReducedFeed("/home/mark/coldstore/gtfs/centro20190826")
	fmt.Printf("%s", serviceIDsByDate(feed, "20190901"))
}
