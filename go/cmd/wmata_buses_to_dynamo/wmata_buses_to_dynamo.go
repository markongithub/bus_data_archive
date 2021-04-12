package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"flag"
	"fmt"
	"github.com/markongithub/bus_data_archive/pkg/bus_positions"
	"io/ioutil"
	"os"
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

type BusPositionReportDynamo struct {
	BusPositionReport
	PartitionKey string
	RangeKey     string
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

func ConvertToDynamoReport(b BusPositionReport, retrievedAt time.Time) BusPositionReportDynamo {
	retrievedAtDate := retrievedAt.Format("2006-01-02")
	return BusPositionReportDynamo{
		BusPositionReport: b,
		PartitionKey:      retrievedAtDate,
		RangeKey:          fmt.Sprintf("%s#%s", b.VehicleID, retrievedAt),
	}
}

//	retrievedAtDate := retrievedAt.Format("2006-01-02")
//	output := &BusPositionReportDynamo{}
//	RetrievedAtDate: retrievedAtDate,
//	RetrievedAt: retrievedAt,
//	VehicleID: b.VehicleID }
//	return output
//	TripID: b.TripID,
//	RouteID: b.RouteID,
//	DirectionNum: b.DirectionNum,
//	DirectionText: b.DirectionText,
//	TripHeadSign: b.TripHeadSign,
//	TripStartTime: b.TripStartTime,
//	TripEndTime: b.TripEndTime,
//	BlockNumber: b.BlockNumber,
//	ReportedAt: b.ReportedAt,
//	Lat: b.Lat,
//	Lon: b.Lon,
//	Deviation: b.Deviation

func main() {
	filename := flag.String("input_file", "", "JSON file with bus data")
	// filename := flag.String("input_file", "", "JSON file with bus data")
	flag.Parse()

	m := ParseFile(*filename)
	CheckInvariant(m)
	reportTime := bus_positions.FileTime(*filename)

	// location, err := time.LoadLocation("US/Eastern")
	// check(err)

	// Initialize a session that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials
	// and region from the shared configuration file ~/.aws/config.
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	tableName := "wmata_bus"

	for _, bp := range m.BusPositions {
		// something something reportTime
		bpd := ConvertToDynamoReport(bp, reportTime)
		av, err := dynamodbattribute.MarshalMap(bpd)
		if err != nil {
			fmt.Println("Got error marshalling map:")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// Create item in table
		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(tableName),
		}

		_, err = svc.PutItem(input)
		if err != nil {
			fmt.Println("Got error calling PutItem:")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Println("Successfully added the report on vehicle " + bpd.VehicleID + " to table " + tableName)
		// snippet-end:[dynamodb.go.load_items.call]
	}
}
