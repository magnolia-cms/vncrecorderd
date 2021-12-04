# Vnc Recorder Daemon

A golang based gRPC service API that connects to a VNC host and records the frames using
`ffmpeg` library and `vnc2video`.

# Features

- [x] Concurrent VNC recordings
- [x] Custom recording file name
- [x] Unique VNC host per request
- [x] VNC encoding format per request
- [x] Custom quality and frame rate  
- [x] Dockerized setup

# Usage

## gRPC code generation

* Copy to the `api/vnc.proto` protobuf file in your project of the language of your choice, and adjust
  the options for the target directory for code generation.
  
* Follow the instructions from `https://grpc.io/docs/` to generate the gRPC client side code from the
  protobuf file.
  
## Service APIs

The `api/vnc.proto` exposes the `VncRecorder` service with three APIs
* `Start` - Starts recording from VNC server
* `Stop` - Stops the recording and saves it in the target file.
* `Remove` - Stops the recording and removes the recording file.

Each of the service APIs accept `VncRequest` as a parameter and returns a `VncResponse` result.

The `VncRequest` message type has four fields - one is required and three are optional:
* `host` - The hostname/IP address where the VNC server for the target recording is running.
* `port` - (Optional) The port on which VNC server is listening. Default value is `5900` or the one set for `VNC_PORT`.
* `fileName` - (Optional) The name of file in which the vnc recording should be saved. If a path is provided, the absolute path would be relative to either `/recordings` or the value of `VNC_RECORDINGS_DIR` if set.
                If file name is not provided, the recording will be saved in `vnc` file.
* `mediaType` - (Optional) The target format of the recording. Defaults to `mp4`.

The `VncResponse` message type has two fields:
* `status` - Returns the status of the VNC recording request. If no errors happen, then a status of `STARTED` for `Start` service request, and `DONE` for `Stop` and `Remove` service request types. On any errors, the status code will be `FAILURE`.
* `message` - Returns a failure message. If no errors happened, then message is empty.

# Running

## Bare metal

* Build the go project by executing `go build`
* Export the environment variables for any customization see Configuration Options below.
* Run the binary executable

## Docker compose setup

* Check out the sample `docker-compose.yaml` file at the root of this project
* Adjust the environment variables in the `.env` file 
* Execute `docker-compose up -d && docker-compose logs -f vncrecorder`

## Configuration Options

The VNC recorder service allows you to customize the below parameters:

* `VNC_PASSWORD` - The password used to authenticate with the VNC server. Default value is `secret`.
* `VNC_PORT` - The port on which the VNC server is listening. Default value is `5900`.
* `GRPC_PORT` - The port on which the VNC recording gRPC service is available. Default value is `3000`. This parameter is useful when
                running in bare metal scenario when you have other services running on port `3000`. When running
                as a docker service, only the exposed port mapping needs to be adjusted.
* `VNC_RECORDINGS_DIR` - The directory where VNC recordings are stored. Default value is `/recordings`. 
* `VNC_FRAME_RATE` -  Number of frames per second the target recording file will contain. Default value is `60`.
* `VNC_CONSTANT_RATE_FACTOR` - Quality of the recording. Default value is `0 - (best)`. Valid range is `0-51` where 0 is the best and 51 is the worst.

