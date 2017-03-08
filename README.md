# cw

Tired of not being able to easily tail your AWS CloudWatch Logs? Give `cw` a go!

It provides commands for:

* list of the available log groups
* tail a given log group/stream

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
