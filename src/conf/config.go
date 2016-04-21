package conf

import (
	"container/list"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"jsonwrapper"
)

const (
	CURRENT_VERSION = "0.2.5"

	DEFAULT_CONF_NAME              = "grandma.conf"
	DEFAULT_BIND_PORT              = "443"
	DEFAULT_GRANDMA_NAME           = "Grandma-Sharon"
	DEFAULT_MSG_START_TYPE         = 100
	DEFAULT_MSG_END_TYPE           = 109
	DEFAULT_QUEUE_LENGTH           = 1000
	DEFAULT_NETWORK_SECRET         = "GrandmaService"
	DEFAULT_REST_SECRET            = "GrandmaSecret"
	DEFAULT_CLUSTER_MODE           = false
	DEFAULT_NETWORK_PORT           = "12345"
	DEFAULT_SCHEDULE_TTL_MAX int64 = 30 * 24 * 60 * 60 * 1000
)

const (
	CONF_QUEUE_LENGTH     = "queue_length"
	CONF_MESSAGE_TYPES    = "msg_type"
	CONF_BIND_PORT        = "bind_port"
	CONF_GRANDMA_NAME     = "name"
	CONF_CLUSTER_MODE     = "cluster_mode"
	CONF_SLAVE_LIST       = "slave_list"
	CONF_REGION_LIST      = "region_list"
	CONF_WS_SLAVE_LIST    = "ws_slave_list"
	CONF_CLUSTER_REGIONS  = "region_list"
	CONF_REST_SIG_SECRET  = "rest_secret"
	CONF_NETWORK_SECRET   = "secret"
	CONF_SCHEDULE_TTL_MAX = "ttl_max"
	CONF_NETWORK_PORT     = "network_port"

	CONF_SMS_ID         = "sms_id"
	CONF_SMS_SECRET     = "sms_secret"
	CONF_EMAIL_SENDER   = "email_sender"
	CONF_EMAIL_SMTP     = "email_smtp"
	CONF_EMAIL_UNAME    = "email_username"
	CONF_EMAIL_PWORD    = "email_password"
	CONF_GCM_APP_SECRET = "gcm_secret"
)

var (
	supported_messages *list.List = nil                  // Indicating supported message types
	path_conf          string     = DEFAULT_CONF_NAME    // Config file path
	bind_port          string     = DEFAULT_BIND_PORT    // Binding port
	grandma_name       string     = DEFAULT_GRANDMA_NAME // Name of the running instance
	cluster_mode       bool       = DEFAULT_CLUSTER_MODE
	queue_length       int        = DEFAULT_QUEUE_LENGTH
	slave_list         *list.List = nil
	region_list        *list.List = nil
	ws_slave_list      *list.List = nil
	ttl_max            int64      = DEFAULT_SCHEDULE_TTL_MAX
	network_port       string     = DEFAULT_NETWORK_PORT
	network_secret     string     = DEFAULT_NETWORK_SECRET
	rest_secret        string     = DEFAULT_REST_SECRET
)

var (
	ErrorInvalidSettings       = errors.New("Invalid settings read from config file")
	ErrorNotSupportMessageType = errors.New("Message type not supported")
	ErrorNoConfigFileFound     = errors.New("Can not find config file")
	ErrorParsingConfigFile     = errors.New("Error parsing config file")
	ErrorUnknownConfigKey      = errors.New("Invalid setting key in config file")
)

func setDefault() {
	supported_messages = list.New()
	for i := DEFAULT_MSG_START_TYPE; i < DEFAULT_MSG_END_TYPE; i++ {
		supported_messages.PushBack(i)
	}
}

func CheckMsgType(msg_type uint) (bool, error) {
	if supported_messages == nil {
		return false, ErrorInvalidSettings
	}

	for e := supported_messages.Front(); e != nil; e = e.Next() {
		if msg_type == e.Value {
			return true, nil
		}
	}

	return false, ErrorNotSupportMessageType
}

func ReadFlags() bool {
	var version bool = false

	flag.StringVar(&path_conf, "c", "", "-c path_to_config_file")
	flag.StringVar(&bind_port, "p", DEFAULT_BIND_PORT, "-p port_to_bind")
	flag.StringVar(&network_port, "np", DEFAULT_NETWORK_PORT, "-np port_to_listen")
	flag.StringVar(&grandma_name, "n", DEFAULT_GRANDMA_NAME, "-n name")
	flag.BoolVar(&version, "v", false, "-v")

	flag.Parse()

	if version {
		fmt.Println("\nGrandma Scheduling Services Version " + CURRENT_VERSION + "\n")
		return false
	}

	return true
}

func GetPort() string {
	fmt.Println("\nGrandma Scheduling Services (" + grandma_name + ") running on port " + bind_port + "...")
	return ":" + bind_port
}

func GetNetworkPort() string {
	return ":" + network_port
}

func GetClusterMode() bool {
	return cluster_mode
}

func GetSlaveList() *list.List {
	return slave_list
}

func GetRegionList() *list.List {
	return region_list
}

func GetNetworkSecret() string {
	return network_secret
}

func GetRestSecret() string {
	return rest_secret
}

func GetQueueLength() int {
	return queue_length
}

func Configure() {
	err := readConfigFromFile()
	if err != nil {
		panic(err)
	}
}

func GetGrandmaName() string {
	return grandma_name
}

func setKey(key string, obj *jsonwrapper.Object) error {
	switch key {
	case CONF_MESSAGE_TYPES:
		data, err := obj.GetInt64Array(CONF_MESSAGE_TYPES)
		if err != nil {
			return err
		}
		supported_messages = list.New()
		for _, e := range data {
			if int(e) > DEFAULT_MSG_END_TYPE || int(e) < DEFAULT_MSG_START_TYPE {
				return ErrorUnknownConfigKey
			}
			supported_messages.PushBack(e)
		}
		break
	case CONF_BIND_PORT:
		if bind_port != DEFAULT_BIND_PORT {
			break
		}
		data, err := obj.GetString(CONF_BIND_PORT)
		bind_port = data
		if err != nil {
			return err
		}
		break
	case CONF_GRANDMA_NAME:
		if grandma_name != DEFAULT_GRANDMA_NAME {
			break
		}
		data, err := obj.GetString(CONF_GRANDMA_NAME)
		grandma_name = data
		if err != nil {
			return err
		}
		break
	case CONF_NETWORK_PORT:
		if network_port != DEFAULT_NETWORK_PORT {
			break
		}
		data, err := obj.GetString(CONF_NETWORK_PORT)
		network_port = data
		if err != nil {
			return err
		}
		break
	case CONF_NETWORK_SECRET:
		data, err := obj.GetString(CONF_NETWORK_SECRET)
		network_secret = data
		if err != nil {
			return err
		}
		break
	case CONF_SCHEDULE_TTL_MAX:
		data, err := obj.GetInt64(CONF_SCHEDULE_TTL_MAX)
		ttl_max = data
		if err != nil {
			return err
		}
		break
	case CONF_SLAVE_LIST:
		data, err := obj.GetStringArray(CONF_SLAVE_LIST)
		slave_list = list.New()
		for _, e := range data {
			slave_list.PushBack(e)
		}
		if err != nil {
			return err
		}
		break
	case CONF_CLUSTER_MODE:
		data, err := obj.GetBoolean(CONF_CLUSTER_MODE)
		cluster_mode = data
		if err != nil {
			return err
		}
		break
	case CONF_QUEUE_LENGTH:
		data, err := obj.GetInt64(CONF_QUEUE_LENGTH)
		if err != nil {
			return err
		}
		if data > 1000000 || data < 10 {
			return ErrorInvalidSettings
		}
		queue_length = int(data)
		break
	case CONF_REST_SIG_SECRET:
		data, err := obj.GetString(CONF_REST_SIG_SECRET)

		if err != nil {
			return err
		}

		rest_secret = data
		break
	default:
		return ErrorUnknownConfigKey
	}
	return nil
}

func readConfigFromFile() error {
	setDefault()

	if len(path_conf) == 0 {
		files, err := ioutil.ReadDir(".")
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.Name() == DEFAULT_CONF_NAME {
				path_conf = DEFAULT_CONF_NAME
				break
			}
		}

		if len(path_conf) == 0 {
			return nil
		}

	}

	content, err := ioutil.ReadFile(path_conf)
	if err != nil {
		return err
	}

	obj, err := jsonwrapper.NewObjectFromBytes(content)
	if err != nil {
		return ErrorParsingConfigFile
	}

	key_map := obj.Map()
	for key := range key_map {
		err := setKey(key, obj)
		if err != nil {
			return ErrorInvalidSettings
		}
	}
	return nil
}
