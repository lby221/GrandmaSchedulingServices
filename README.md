# Grandma Scheduling Services (GSS)

GSS is a distributed scheduling service helping you to publish multiple types of messages across regions without runtime delays. This project is still under development. The current version is 0.2.5.

### Update Notes

#### 0.2.6 (working)

1. Added config settings for queues, rest call signatures and messages.

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

```golang
func GetUser(userTok string) *jsonwrapper.Object {

	obj, err := sellyxrequest.JsonHttpRequest("GET", "auth.sellyx.com", "/auth/validate", "key="+userTok, "")

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