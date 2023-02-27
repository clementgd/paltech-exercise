package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	. "paltech.robot/robot"
)




type GardenArea struct {
	Name         string
	MinLatitude  float64
	MaxLatitude  float64
	MinLongitude float64
	MaxLongitude float64
}

// https://maps.google.com/?q=<lat>,<lng>
var gardenArea = &GardenArea{
	Name:         "Südliche Fröttmaninger Heide",
	MinLatitude:  48.210965,
	MaxLatitude:  48.224528,
	MinLongitude: 11.599042,
	MaxLongitude: 11.614783,
}

const GlobalTimeMultiplier float64 = 10
const MinUpdateFrequencySeconds int = 60
const MaxUpdateFrequencySeconds int = 120

const DistanceLatitudeDivider float64 = 110574
const DistanceLongitudeDivider float64 = 111320
const MaxDirectionChangeRadians float64 = math.Pi / 4
const MinForwardSpeed float64 = 0.1
const MaxForwardSpeed float64 = 3
const WaypointReachedProbability float64 = 0.4
const DefaultWaypointsTotal = 20

var wg sync.WaitGroup



func degreeToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func radiansToDegree(radians float64) float64 {
	return radians * 180 / math.Pi
}

func distanceNorthToLatitudeChange(distanceMeters float64) float64 {
	return distanceMeters / DistanceLatitudeDivider
}

func distanceEastToLongitudeChange(distanceMeters float64, latitude float64) float64 {
	return distanceMeters / (DistanceLongitudeDivider * math.Cos(degreeToRadians(latitude)))
}

func getHypotenuse(x float64, y float64) float64 {
	return math.Sqrt(math.Pow(x, 2) + math.Pow(y, 2))
}

func getRandomDirectionChange() float64 {
	return randFloat64(-MaxDirectionChangeRadians, MaxDirectionChangeRadians)
}

func ensureBounds(robotStatus *RobotStatus, area *GardenArea) {
	if robotStatus.Latitude < area.MinLatitude {
		robotStatus.Latitude = area.MinLatitude
		robotStatus.InvertSpeedNorth()
	} else if robotStatus.Latitude > area.MaxLatitude {
		robotStatus.Latitude = area.MaxLatitude
		robotStatus.InvertSpeedNorth()
	}

	if robotStatus.Longitude < area.MinLongitude {
		robotStatus.Longitude = area.MinLongitude
		robotStatus.InvertSpeedEast()
	} else if robotStatus.Longitude > area.MaxLongitude {
		robotStatus.Longitude = area.MaxLongitude
		robotStatus.InvertSpeedEast()
	}
}

func generateNextRobotStatus(robot *Robot, newTimestamp int64) *RobotStatus {
	var nextRobotStatus RobotStatus = *robot.StatusHistory[len(robot.StatusHistory)-1]
	elapsedSeconds := float64(newTimestamp - nextRobotStatus.Timestamp)
	distanceNorth := nextRobotStatus.GetSpeedNorth() * elapsedSeconds
	distanceEast := nextRobotStatus.GetSpeedEast() * elapsedSeconds

	nextRobotStatus.Latitude += distanceNorthToLatitudeChange(distanceNorth)
	nextRobotStatus.Longitude += distanceEastToLongitudeChange(distanceEast, nextRobotStatus.Latitude)
	nextRobotStatus.DistanceCovered += getHypotenuse(distanceNorth, distanceEast)
	if randFloat64(0, 1) <= WaypointReachedProbability {
		nextRobotStatus.WaypointsReached += 1
		nextRobotStatus.WaypointsSuccessful += 1
	}

	nextDirection := nextRobotStatus.GetDirectionAngleNorth() + getRandomDirectionChange()
	nextForwardSpeed := randFloat64(MinForwardSpeed, MaxForwardSpeed)
	nextRobotStatus.SetDirectionAndForwardSpeed(nextDirection, nextForwardSpeed)
	ensureBounds(&nextRobotStatus, gardenArea)

	nextRobotStatus.Timestamp = newTimestamp

	return &nextRobotStatus
}

func getInitialStatus(timestamp int64) *RobotStatus {
	robotStatus := new(RobotStatus)
	robotStatus.Timestamp = timestamp
	robotStatus.Latitude = randFloat64(gardenArea.MinLatitude, gardenArea.MaxLatitude)
	robotStatus.Longitude = randFloat64(gardenArea.MinLongitude, gardenArea.MaxLongitude)

	forwardSpeed := randFloat64(MinForwardSpeed, MaxForwardSpeed)
	direction := randFloat64(0, 2 * math.Pi)
	robotStatus.SetDirectionAndForwardSpeed(direction, forwardSpeed)

	robotStatus.WaypointsTotal = DefaultWaypointsTotal
	return robotStatus
}

func getCurrentTimeMultiplied(initialTime int64) int64 {
	return initialTime + int64((float64(time.Now().Unix()) - float64(initialTime)) * GlobalTimeMultiplier)
}




func getStaticMapUrl(robot *Robot) string {
	url := "https://maps.googleapis.com/maps/api/staticmap?size=1000x1000&&path=color:0xff0000ff|weight:1|"
	url += robot.GetPath()
	url += "&sensor=false&key=MAPSKEY"
	return url
}

func requestCreateRobot(initialStatus *RobotStatus) int {
	postBody, _ := json.Marshal(*initialStatus)
	responseBody := bytes.NewBuffer(postBody)
	response, err := http.Post("http://127.0.0.1:1323/register-robot", "application/json", responseBody)
	if err != nil {
        log.Fatal(err)
    }

	var returnedId int
	json.NewDecoder(response.Body).Decode(&returnedId)
	fmt.Println("Created robot with id : ", returnedId)
	return returnedId
}

func requestUpdateRobot(status *RobotStatus, robotId int) {
	postBody, _ := json.Marshal(*status)
	url := "http://127.0.0.1:1323/update-robot/" + strconv.Itoa(robotId)
	_, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
        log.Fatal(err)
    }
}


func simulateRobot() {
	defer wg.Done()

	currentTimestamp := time.Now().Unix()
	robot := new(Robot)
	initialStatus := getInitialStatus(currentTimestamp)
	robot.Id = requestCreateRobot(initialStatus)
	robot.AppendStatus(initialStatus)
	waypointsTotal := initialStatus.WaypointsTotal
	nIterations := 0

	for robot.GetLatestStatus().WaypointsReached < waypointsTotal {
		fmt.Println(
			"Robot", robot.Id, 
			"waypoints reached : ", robot.GetLatestStatus().WaypointsReached, "/", waypointsTotal,
			"on", nIterations, "iterations",
		)
		nextUpdateDelaySeconds := randInt(MinUpdateFrequencySeconds, MaxUpdateFrequencySeconds)
		sleepTime := time.Duration( float64(nextUpdateDelaySeconds * 1000) / GlobalTimeMultiplier ) * time.Millisecond
		fmt.Println("Sleep time : ", sleepTime)
		time.Sleep(sleepTime)
		
		currentTimestamp += int64(nextUpdateDelaySeconds)
		nextRobotStatus := generateNextRobotStatus(robot, currentTimestamp)
		requestUpdateRobot(nextRobotStatus, robot.Id)
		robot.StatusHistory = append(robot.StatusHistory, nextRobotStatus)
		nIterations++
	}
}

func simulateNRobots(n int) {
	wg.Add(n)

	for i := 0 ; i < n ; i++ {
		go simulateRobot()
		time.Sleep(4 * time.Second)
	}

	wg.Wait()
}

func testPathGeneration() {
	currentTimestamp := time.Now().Unix()
	robot := new(Robot)
	initialStatus := getInitialStatus(currentTimestamp)
	robot.AppendStatus(initialStatus)
	waypointsTotal := initialStatus.WaypointsTotal
	nIterations := 0

	for robot.GetLatestStatus().WaypointsReached < waypointsTotal {
		fmt.Println(
			"Robot", robot.Id, ":",
			robot.GetLatestStatus().WaypointsReached, "/", waypointsTotal, "waypoints reached, ",
			nIterations, "iterations",
		)
		nextUpdateDelaySeconds := randInt(MinUpdateFrequencySeconds, MaxUpdateFrequencySeconds)
		sleepTime := time.Duration( float64(nextUpdateDelaySeconds * 1000) / GlobalTimeMultiplier ) * time.Millisecond
		fmt.Println("Next delay : ", nextUpdateDelaySeconds, "s")
		fmt.Println("Sleep time : ", sleepTime)
		time.Sleep(sleepTime)
		
		fmt.Println("\n============================================================================")
		currentTimestamp += int64(nextUpdateDelaySeconds)
		nextRobotStatus := generateNextRobotStatus(robot, currentTimestamp)
		robot.StatusHistory = append(robot.StatusHistory, nextRobotStatus)
		fmt.Println("Static map url : ", getStaticMapUrl(robot))
		nIterations++
	}

	// robot.GenerateAndSavePathImage()
}


func main() {
	rand.Seed(time.Now().UnixNano())

	// testPathGeneration()
	// simulateRobot()
	simulateNRobots(1)
}

func randFloat64(min float64, max float64) float64 {
	return min + rand.Float64() * (max - min)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max - min + 1)
}
