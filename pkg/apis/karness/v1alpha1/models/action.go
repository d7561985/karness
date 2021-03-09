package models

//
type Event struct {
	Action    Action            `json:"action"`
	Condition CompleteCondition `json:"condition"`
}

type Action struct {
}

type CompleteCondition struct {
}
