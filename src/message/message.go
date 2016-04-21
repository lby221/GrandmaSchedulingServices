package message

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

type Obj struct {
	MessageType int
	Endpoint    string
	MessageBody string
	Expiration  int64
}

// Message type list
const (
	S_DELETE_MESSAGE         = 100
	S_APNS_NOTIFICATION      = 101
	S_GCM_NOTIFICATION       = 102
	S_BAIDU_NOTIFICATION     = 103
	S_TOPIC_NOTIFICATION     = 104
	S_WEBSOCKET_NOTIFICATION = 105
	S_EMAIL_NOTIFICATION     = 106
	S_REST_NOTIFICATION      = 107
	S_RABBITMQ_NOTIFICATION  = 108 // todo
	S_SMS_NOTIFICATION       = 109
)

// Error list
var (
	ErrorInvalidType           = errors.New("Type not exists")
	ErrorExpirationTooBig      = errors.New("Expiration time too big")
	ErrorNegativeExpiration    = errors.New("Negative expiration time")
	ErrorNoEndpoint            = errors.New("No endpoint provided")
	ErrorAlgorithmNotSupported = errors.New("Algorithm not supported")
)

// Hashing algorithm list
const (
	S_ALG_SHA1   = 201
	S_ALG_SHA256 = 202
	S_ALG_MD5    = 203
)

func NewMessageObject(msg_type int, endpoint string, msg_body string, exp_time int64) (*Obj, error) {
	if len(endpoint) < 1 {
		return nil, ErrorNoEndpoint
	} else if exp_time > 90*24*60*60*1000 {
		return nil, ErrorExpirationTooBig
	} else if exp_time < 0 {
		return nil, ErrorNegativeExpiration
	} else if msg_type > 109 || msg_type < 100 {
		return nil, ErrorInvalidType
	}

	return &Obj{msg_type, endpoint, msg_body, exp_time}, nil
}

// func (o *obj) CreateHashedSchedule(algorithm int) {
// 	hashed_msg = o.GetMessageHashing(algorithm)

// }

func (o *Obj) GetMessageHashing(algorithm int) (string, error) {
	if algorithm == S_ALG_MD5 {
		return getMD5Hash(o.Endpoint + "\n" + o.MessageBody), nil
	} else if algorithm == S_ALG_SHA1 {
		return getSHA1Hash(o.Endpoint + "\n" + o.MessageBody), nil
	} else if algorithm == S_ALG_SHA256 {
		return getSHA256Hash(o.Endpoint + "\n" + o.MessageBody), nil
	} else {
		return "", ErrorAlgorithmNotSupported
	}
}

func (o *Obj) GetMessagePayload() []byte {
	return []byte(strconv.Itoa(o.MessageType) + "\n" +
		o.Endpoint + "\n" + o.MessageBody + "\n" + strconv.FormatInt(o.Expiration, 10))
}

func NewMessageFromPayload(payload []byte) (*Obj, error) {
	message_str := strings.Split(string(payload), "\n")
	log.Println(string(message_str[0]))
	log.Println(string(message_str[1]))
	log.Println(string(message_str[2]))
	log.Println(string(message_str[3]))
	msg_type, _ := strconv.Atoi(message_str[0])
	msg_exp, _ := strconv.ParseInt(message_str[3], 10, 64)
	return NewMessageObject(msg_type, message_str[1],
		message_str[2], msg_exp)
}

// func (o *obj) SendMessage() {

// }
