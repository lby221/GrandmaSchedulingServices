# Grandma Scheduling Services (GSS)

[![Build Status](https://travis-ci.org/lby221/GrandmaSchedulingServices.svg?branch=master)](https://travis-ci.org/lby221/GrandmaSchedulingServices)

GSS is a distributed scheduling service helping you to publish multiple types of messages across regions without runtime delays. This project is still under development. The current version is 0.2.5.

### Usage

#### Get The Code

Clone or download the code from github and unzip (if necessary)

#### Install GO

Please make sure you have already installed Go compiler to your computer. Either gccGo or cgo will work.
Check [Go Installation Intructions](https://golang.org/doc/install) for more details.

#### Set Environment Variables

For windows, go to environment path manager and set add GOPATH to your path to GSS.
For Linux/Mac, set the environment variables in terminal:
```bash
export GOPATH=/path/to/GSS
```
You can also add the above to your bashrc file or profile file (Linux and Mac)

#### Compile

1. cd to your path to GSS in your terminal.
2. Type the following to compile GSS:
```bash
go build -o gss server.go routes.go
```

#### Verify Compilation

```bash
./gss -v
```
Current version number will be printed if you have successfully compiled GSS.

#### Run GSS

```bash
./gss
```
If you prefer, you can add a link to your system's bin folder.


### Update Notes

#### 0.2.6 (working)

1. Added config settings for queues, rest call signatures and messages.
2. Support big data distributions with third party software execution in cluster.
3. Fixed a bug that schedules may not be delivered immediately.

#### 0.2.5 (current)

1. Fixed a bug that the order of messages sent to slaves switched upon random network failures.
2. Fixed a bug that GSS will crash after restarting and reconnecting (mutex locks and condition variables). 
3. Added support for restarting master node upon crash or network failure.
4. Added support for reconnecting master node from slave and heartbeat.

#### 0.2.4

1. Fixed a bug that messages sent to slaves may be cut into multiple pieces.
2. Support slave nodes reconnecting to master and sending/receiving heartbeat signals.
3. Support message recovery for slave and master nodes after respawn.

#### 0.2.3

1. Schedule information stored on disk.
2. Separate database instances for different nodes.

... For more please check update logs.

### Brief introduction to codes

##### 1. Change routes.go if you want to support more routes. 

Add a new handler to implement your API. You can turn this scheduler into an API server.

##### 2. Websocket needs authentication to work. 

You can add authorization to GSS to active this functionality. Assume your user verifying package is called "user", then you can build functions like the following and change the code in routes.go to support websocket.

```go
func GetUser(userTok string) *jsonwrapper.Object {

	obj, err := request.JsonHttpRequest("GET", "auth.example.com", "/auth/validate", "key="+userTok, "")

	if err != nil || !obj.Success {
		return nil
	} else {
		u, _ := obj.Json.GetObject("user")
		return u
	}
}
```

##### 3. config.go needs more work to do. 

Feel free to add more options to config.

### API Documentations

##### API Authentication

To verify the source of the caller and keep the API secure, GSS uses a REST based signature authentication (Currently GSS doesn't support authentication using OAuth by default, you can feel free to modify routes.go to support your own way to do authorization). To finish the authorization process, please refer to the following steps:

1. Set up REST secret in both your caller project and grandma.conf

	grandma.conf:
	```json
	{
		...
		"rest_secret" : "jU79$ifK9*du|d-9s",
		...
	}
	```

2. Build string to sign in your caller program based on your request

	You should build your string to sign according to this format:
	```
	string_to_sign = <Request-Method> + '\n' + <Current-Epoch-Time-In-Milliseconds> + '\n' + <Request-Body>
	```
	Note: ```'\n'``` is the escape for newline

	For example, a valid string to sign is like
	```
	POST\n1461721658441\n{"msg":"I_LOVE_MY_GRANDMA"}
	```
	Note: In some cases, double qoute should be represented as its escape ```'\"'```

3. Sign your string to sign

	Generate the signature using HMAC-SHA256 with the REST secret you set in grandma.conf. The output signature's encoding method should be set to hex string.

	The following is the node.js example showing how to make the signature:
	```javascript
	var hash = crypto.createHmac('sha256', rest_secret).update(string_to_sign).digest('hex');

	```

4. Turn your hex string into Base64 and delete all equal signs

	A valid signature should be like (using string to sign and secret from above examples)
	```
	MTRmODFlZWMwZTk2NDVhNDUzYzM5NmIzNjkwN2FiODgxMDlhY2IzY2NhNDMwNmMyODBiMmI3NjM1NWY2MmVjMg
	```

5. Build your request URL

	Your request url should be of following format:
	```
	http(s)://host_of_gss:port/?time=<SAME_TIME_IN_STRING_TO_SIGN>&token=<SIGNATURE>
	```
	For example, if your GSS is running at https://gss.example.com:8090/, your request url should be:
	```
	https://gss.example.com:8090/?time=1461721658441&token=MTRmODFlZWMwZTk2NDVhNDUzYzM5NmIzNjkwN2FiODgxMDlhY2IzY2NhNDMwNmMyODBiMmI3NjM1NWY2MmVjMg
	```

6. Test

	Try making a call with a body of JSON and request url you generated from the previous steps:
	```json
	{
		"msg":"I_LOVE_MY_GRANDMA"
	}
	```
	You will get the following if you set everything correctly:
	```json
	{
		"msg":"I_LOVE_YOU"
	}
	```
