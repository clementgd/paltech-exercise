package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	. "paltech.robot/robot"
	. "paltech.telegram_bot/telegram_bot"
)

// go mod edit -replace paltech.telegram_bot/telegram_bot=../telegram_bot
// go mod edit -replace paltech.robot/robot=../robot

const RobotUpdateTimeoutMilliseconds = 10500
const RobotPeriodicUpdatesIntervalSeconds = 20

var telegramBot *TelegramBot
var robots = make([]*Robot, 0, 5)
var robotsUpdateTimers = make([]*time.Timer, 0, 5)
var robotsTimeoutTimers = make([]*time.Timer, 0, 5)
var hasRobotTimedOut = make([]bool, 0, 5)
var robotsMutex sync.Mutex
var nextRobotId = 0

func main() {
	e := echo.New()
	e.POST("/register-robot", registerRobot)
	e.POST("/update-robot/:id", updateRobot)
	e.GET("/path/:filename", getPathImage)

	var err error
	telegramBot, err = NewTelegramBot("TELEGRAMKEY")
    if err != nil {
        log.Panic(err)
		return
    }
	telegramBot.Robots = &robots
	telegramBot.RobotsMutex = &robotsMutex

	telegramBot.ListenAndServe()
	e.Logger.Fatal(e.Start(":1323"))
}

func parseId(c echo.Context) (int, error) {
	strId := c.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "Robot ID must be an integer")
	}
	return id, nil
}

func timeout(robotId int) {
	<-robotsTimeoutTimers[robotId].C
	fmt.Println("\nRobot", robotId, "timed out")
	if robotsUpdateTimers[robotId] != nil {
		robotsUpdateTimers[robotId].Stop()
	}
	// TODO Maybe add synchronization mechanics here, per robot
	hasRobotTimedOut[robotId] = true
	telegramBot.SendTimeoutMessage(robotId)
}

func periodicUpdate(robotId int) {
	<-robotsUpdateTimers[robotId].C
	telegramBot.SendPeriodicUpdateForRobot(robotId, false)
	resetPeriodicUpdateTimerForRobot(robotId)
}

func resetTimeoutTimerForRobot(robotId int) {
	robotsTimeoutTimers[robotId].Stop()
	hasRobotTimedOut[robotId] = false
	robotsTimeoutTimers[robotId] = time.NewTimer(time.Duration(RobotUpdateTimeoutMilliseconds) * time.Millisecond)
	fmt.Println("Reset timeout timer for robot", robotId)
	go timeout(robotId)
}

func resetPeriodicUpdateTimerForRobot(robotId int) {
	robotsUpdateTimers[robotId] = time.NewTimer(time.Duration(RobotPeriodicUpdatesIntervalSeconds) * time.Second)
	go periodicUpdate(robotId)
}


func registerRobot(c echo.Context) error {
	robot := new(Robot)

	parsedInitialStatus := new(RobotStatus)
	if err := c.Bind(parsedInitialStatus); err != nil {
		return err
	}

	robotId := nextRobotId
	nextRobotId++
	robot.Id = robotId
	robot.AppendStatus(parsedInitialStatus)

	robotsMutex.Lock()
	robots = append(robots, robot)
	
	// fmt.Println("Created robot : ", robot.ToString())
	robotsUpdateTimers = append(robotsUpdateTimers, nil)
	robotsTimeoutTimers = append(
		robotsTimeoutTimers, 
		time.NewTimer(time.Duration(RobotUpdateTimeoutMilliseconds) * time.Millisecond),
	)
	hasRobotTimedOut = append(hasRobotTimedOut, false)
	// Releasing the mutex after having created the timer entries to make we have sync
	// between robots, robotsUpdateTimers, robotsTimeoutTimers and hasRobotTimedOut indexes
	robotsMutex.Unlock()

	go timeout(robotId)
	fmt.Println("\nRegistered robot", robotId)
	return c.JSON(http.StatusOK, robot.Id)
}

func updateRobot(c echo.Context) error {
	id, httpErr := parseId(c)
	if httpErr != nil {
		return httpErr
	}

	parsedStatus := new(RobotStatus)
	if err := c.Bind(parsedStatus); err != nil {
		return err
	}

	robotsMutex.Lock()
	if id >= len(robots) {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	robots[id].AppendStatus(parsedStatus)
	var robotCopy Robot = *robots[id]
	robotsMutex.Unlock()

	fmt.Println("\nUpdated robot :", id)
	result := c.String(http.StatusOK, "")
	robotCopy.GenerateAndSavePathImage()

	if hasRobotTimedOut[id] {
		fmt.Println("Robot", id, "came back online")
		telegramBot.SendPeriodicUpdateForRobot(id, true)
		resetPeriodicUpdateTimerForRobot(id)
	}
	if robotsUpdateTimers[id] == nil {
		resetPeriodicUpdateTimerForRobot(id)
	}
	resetTimeoutTimerForRobot(id)
	return result
}

func getPathImage(c echo.Context) error {
	filepath := "pathImages/" + c.Param("filename")
	return c.File(filepath)
}
