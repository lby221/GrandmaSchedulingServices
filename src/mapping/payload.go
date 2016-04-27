package mapping

import (
	"errors"
	"jsonwrapper"
	"net/url"
)

var (
	ErrInvalidPayload   = errors.New("Invalid payload")
	ErrWrongPayloadSize = errors.New("Payload size doesn't match petition size")
)

type Payload struct {
	urldata bool
	url     string
	data    string
}

func getPayload(payload *jsonwrapper.Object, size int) ([]*Payload, error) {
	t, err := payload.GetString("type")

	if err != nil {
		return nil, err
	}

	switch t {
	case "single_file":
		var p *Payload
		urlpath, err := payload.GetString("url")

		if err != nil {
			return nil, err
		}

		if _, err := url.Parse(urlpath); err != nil {
			return nil, ErrInvalidPayload
		}
		p.url = urlpath
		p.urldata = true

		return processSinglePayload(p, size)
	case "separate_file":
		return processSeparatePayload(payload, size, true)
	case "single_data":
		var p *Payload
		data, err := payload.GetString("data")

		if err != nil {
			return nil, err
		}

		if data == "" {
			return nil, ErrInvalidPayload
		}

		p.data = data
		p.urldata = false

		return processSinglePayload(p, size)
	case "separate_data":
		return processSeparatePayload(payload, size, false)
	default:
		return nil, ErrInvalidPayload
	}
}

func processSinglePayload(p *Payload, size int) ([]*Payload, error) {
	if p == nil {
		return nil, ErrInvalidPayload
	}

	payloads := make([]*Payload, size)

	for i := 0; i < size; i++ {
		payloads[i] = p
	}

	return payloads, nil
}

func processSeparatePayload(p *jsonwrapper.Object, size int, isurl bool) ([]*Payload, error) {
	if p == nil {
		return nil, ErrInvalidPayload
	}

	payloads := make([]*Payload, size)

	json_payloads, err := p.GetStringArray("payloads")

	if err != nil {
		return nil, err
	}

	if len(json_payloads) != size {
		return nil, ErrWrongPayloadSize
	}

	for i := 0; i < size; i++ {
		payloads[i] = new(Payload)
		payloads[i].urldata = isurl

		if isurl {
			if _, err := url.Parse(json_payloads[i]); err != nil {
				return nil, ErrInvalidPayload
			}
			payloads[i].url = json_payloads[i]
		} else {
			payloads[i].data = json_payloads[i]
		}
	}

	return payloads, nil

}
