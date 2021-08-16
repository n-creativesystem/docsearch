package fsm

type Operation int

const (
	Update Operation = iota + 1
	Delete
)

type CommandPayload struct {
	Operation Operation
	Data      map[string]interface{}
}
