# cw

The best way to tail AWS Cloudwatch Logs

It offers commands for:
* list all the log groups
* list all the log stream within a log group
* tail a given log group/log stream

Examples:
* list of the available log groups
  * `cw ls groups`
* list of the log streams in a given log group
  * `cw ls streams my-log-group`
* tail a given log group/stream
  * `cw tail -f my-log-group` 
  * `cw tail -f my-log-group my-log-stream-prefix` 
  * `cw tail my-log-group my-log-stream-prefix 2017-01-01T08:10:10 2017-01-01T08:05:00`  
  * `cw tail -f my-log-group \* 9:00 9:01` The use of the \* wildchar will let you tail all the log streams in my-log-group. 

`cw` uses the default credentials profile(stored in ./aws/credentials) for authentication and shared config(.aws/config) for target AWS region. Time and dates are always treated in UTC.
 
## Installation

On Mac OSX:

* `brew tap lucagrulla/cw`
* `brew install cw`

Using go tools:

`go get github.com/lucagrulla/cw`

## TODOs:
* throttle AWS API request so that not to exceed rate limit
* ~~make the usage of log group+leg stream easier~~
* ~~fix bug for long polling once events are finished(currently we print again a last chunk of alerts)~~
* ~~add an optionl end date for time window~~
* ~~allow more flexible startTime format(no seconds means 00, no minutes means 00:00)~~
* ~~add coloured output~~
* ~~add brew recipe~~
