# DRAS — Doppler Radar Notification Service
 
 This programs monitors either a single, or multiple, WSR-88D sites and sends alerts via Pushover based on change in status.

 ## How To Use

 ### Requirements

 - Pushover Account
 - A device with the pushover mobile application installed

 ### Binary Method

 1. Assuming you have Go installed on your system, head over to the Github [Releases](https://github.com/jacaudi/dras/releases) and grab the URL to the latest release. 
 2. Run go install <URL> (e.g. `go install github.com/jacaudi/dras@v1.0.0`)
 3. Be sure to set the following Environmental Variables
    - `STATION_IDS` — WSR-88D (Radar) Sites (e.g. KRAX - Raleigh/Durham)
    - `PUSHOVER_USER_KEY` — Your Pushover User Key
    - `PUSHOVER_API_TOKEN` — Your Pushover API Token
 4. Enjoy! 


 ### Standalone Container Method

 ```
 docker pull ghcr.io/jacaudi/dras:v1

 docker run -d \
   -e STATION_IDS=KRAX \
   -e PUSHOVER_USER_KEY=<KEY> \
   -e PUSHOVER_API_TOKEN=<TOKEN> \
   ghcr.io/jacaudi/dras:v1
 ```

 ### Kubernetes Method

 See the [kubernetes](examples/kubernetes.yaml) file in [examples](examples) folder — It contains an example deployment, configmap, and secret.

## How To Contribute

This project welcomes any feature improvements or bugs found via PRs. Thank you!

## License

[MIT](LICENSE)