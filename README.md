# Crypto Coin Spike Alerter

A [Slack alerter](https://github.com/natoen/duckcoyn/blob/0f99beca52cf048d866319a40f089581e86455d2/helpers/helpers.go#L134) written in Go whenever there is a spike in Binance. It will run through all the coins in parallel every minute and do an alert once a day of the specific coins when caught. The criteria of a spike is self made but mostly about volume, price, inconsistency, and anomaly. If you are looking at the commits, there was DB migrations,
testings, and output files here before but ultimately this became just a hobby to
see if volume base anomaly will cause a spike. No profit was made during this hobby.

### How to run

- you should have Go installed
- `go run main.go`

### Others

Comments were added here at the last minute for the readers to make it
friendlier. The keys committed here before are all disabled. This repo will go
back into a private after some time.
