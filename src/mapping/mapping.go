package mapping

import (
	"io"
	"jsonwrapper"
)

type MappingCall struct {
	Payloads    []*Payload
	Resorg      int
	Trigger_Mtd int
}

type Mapping interface {
}

func ParseMapping(content io.Reader) (*MappingCall, error) {
	json, err := jsonwrapper.NewObjectFromReader(content)

	if err != nil {
		return nil, err
	}

	num_petition, err := json.GetInt64("num_petition")

	if err != nil {
		return nil, err
	}

	payload_obj, err := json.GetObject("payload")

	if err != nil {
		return nil, err
	}

	payloads, err := getPayload(payload_obj, int(num_petition))

	if err != nil {
		return nil, err
	}

	result_method, err := getResultOrgMethod(json)

	if err != nil {
		return nil, err
	}

	trigger_method, err := getTriggerMethod(json)

	if err != nil {
		return nil, err
	}

	return createMappingCall(payloads, result_method, trigger_method), nil

}

func getResultOrgMethod(json *jsonwrapper.Object) (int, error) {
	r, err := json.GetString("result")

	if err != nil {
		return -1, err
	}

	switch r {
	case "none":
		return 0, nil
	case "combined":
		return 1, nil
	case "separated":
		return 2, nil
	default:
		return 0, nil
	}
}

func getTriggerMethod(json *jsonwrapper.Object) (int, error) {
	t, err := json.GetString("trigger")

	if err != nil {
		return -1, err
	}

	switch t {
	case "none":
		return 100, nil
	case "email":
		return 106, nil
	case "sms":
		return 109, nil
	case "rest":
		return 107, nil
	case "push":
		return 102, nil
	default:
		return 100, nil
	}
}

func getOnErrorMethod(json *jsonwrapper.Object) (int, error) {
	e, err := json.GetString("on_error")

	if err != nil {
		return -1, err
	}

	switch e {
	case "terminate":
		return 0, nil
	case "continue":
		return 1, nil
	default:
		return 0, nil
	}
}

func createMappingCall(payloads []*Payload, result int, trigger int) *MappingCall {
	call := new(MappingCall)

	call.Payloads = payloads
	call.Resorg = result
	call.Trigger_Mtd = trigger

	return call
}
