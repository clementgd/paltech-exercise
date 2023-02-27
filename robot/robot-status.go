package robot

import (
	"math"
)

type RobotStatus struct {
	Timestamp           int64      `json:"timestamp"`
	Latitude            float64    `json:"lat"`
	Longitude           float64    `json:"lon"`
	OdometerSpeed       [3]float64 `json:"odom_speed"`
	DistanceCovered     float64    `json:"distance_covered"`
	WaypointsReached    int        `json:"waypoints_reached"`
	WaypointsSuccessful int        `json:"waypoints_successful"`
	WaypointsTotal      int        `json:"waypoints_total"`
}

func (robotStatus *RobotStatus) GetSpeedNorth() float64 {
	return robotStatus.OdometerSpeed[0]
}

func (robotStatus *RobotStatus) GetSpeedEast() float64 {
	return robotStatus.OdometerSpeed[1]
}

func (robotStatus *RobotStatus) GetDirectionAngleNorth() float64 {
	return math.Atan2(robotStatus.GetSpeedNorth(), robotStatus.GetSpeedEast())
}

func (robotStatus *RobotStatus) SetSpeedNorth(speed float64) {
	robotStatus.OdometerSpeed[0] = speed
}

func (robotStatus *RobotStatus) SetSpeedEast(speed float64) {
	robotStatus.OdometerSpeed[1] = speed
}

func (robotStatus *RobotStatus) SetDirectionAndForwardSpeed(angleNorthRadians float64, forwardSpeed float64) {
	robotStatus.SetSpeedNorth(forwardSpeed * math.Sin(angleNorthRadians))
	robotStatus.SetSpeedEast(forwardSpeed * math.Cos(angleNorthRadians))
}

func (robotStatus *RobotStatus) InvertSpeedNorth() {
	robotStatus.OdometerSpeed[0] = -robotStatus.OdometerSpeed[0]
}

func (robotStatus *RobotStatus) InvertSpeedEast() {
	robotStatus.OdometerSpeed[1] = -robotStatus.OdometerSpeed[1]
}