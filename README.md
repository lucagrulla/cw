# cw

A simpler way to tail AWS Cloudwatch Logs

It provides commands for:

* list of the available log groups
  * `cw ls`
* tail a given log group/stream
  * `cw tail my-log-group 2017-01-01T08:10:10 2017-01-01T08:05:00`
  * `cw tail -f my-log-group` 
  * `cw tail -f my-log-group 9:00 9:01`

`cw` uses the default credentials profile(stored in ./aws) to authenticate against AWS.
 
## Installation

On Mac OSX:

* `brew tap lucagrulla/cw`
* `brew install cw`

Using go tools:

`go get github.com/lucagrulla/cw`

## TODOs:

* throttle AWS API request so that not to exceed rate limit
* ~~fix bug for long polling once events are finished(currently we print again a last chunk of alerts)~~
* ~~add an optionl end date for time window~~
* ~~allow more flexible startTime format(no seconds means 00, no minutes means 00:00)~~
* ~~add coloured output~~
* ~~add brew recipe~~
