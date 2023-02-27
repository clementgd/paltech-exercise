package robot

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)


type Robot struct {
	Id            int
	StatusHistory []*RobotStatus
}

// Function is exported only if it starts with uppercase
func (robot *Robot) AppendStatus(robotStatus *RobotStatus) { 
	robot.StatusHistory = append(robot.StatusHistory, robotStatus)
}

func (robot *Robot) GetLatestStatus() *RobotStatus {
	return robot.StatusHistory[len(robot.StatusHistory) - 1]
}

func (robot *Robot) ToString() string {
	str := "Robot Object\n"
	str += "id : " + strconv.Itoa(robot.Id) + "\n"
	for _, status := range robot.StatusHistory {
		str += fmt.Sprintf("%#v", *status) + "\n"
	}
	return str
}

func (robot *Robot) GetPath() string {
	pathString := ""
	for i, status := range robot.StatusHistory {
		pathString += strconv.FormatFloat(status.Latitude, 'f', 6, 64)
		pathString += ","
		pathString += strconv.FormatFloat(status.Longitude, 'f', 6, 64)

		if i < len(robot.StatusHistory) - 1 {
			pathString += "|"
		}
	}
	return pathString
}

func (robot *Robot) GenerateAndSavePathImage() {
	statusHistoryLength := len(robot.StatusHistory)
	url := "https://maps.googleapis.com/maps/api/staticmap?size=1000x1000&zoom=15&center=48.218885,11.607754&path=color:0xff0000ff|weight:1|"
	url += robot.GetPath()
	url += "&sensor=false&key=MAPSKEY"

	filepath := "pathImages/path-" + strconv.Itoa(robot.Id) + "-" + strconv.Itoa(statusHistoryLength) + ".png"
    file, err := os.Create(filepath)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

	response, e := http.Get(url)
    if e != nil {
        log.Fatal(e)
    }
    defer response.Body.Close()

    _, err = io.Copy(file, response.Body)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Downloaded path image at : ", filepath)
}