# cw

Tired of not being able to easily tail your AWS CloudWatch? Give cw a go!

cw it's a CLI tool for an easier interaction with AWS CloudWatch. 

It provides commands for:

* tail a given log group/stream
* list of the available log groups

`cw` uses the default credentials profile(stored in .aws) to authenticate against AWS.
 
## Installation

On Mac OSX:

* `brew tap lucagrulla/cw`
* `brew install cw`

Using go tools:

`go get github.com/lucagrulla/cw`

## TODOs:

* ~~fix bug for long polling once events are finished(currently we print again a last chunk of alerts)~~
* ~~add an optionl end date for time window~~
* ~~allow more flexible startTime format(no seconds means 00, no minutes means 00:00)~~
* add coloured output
* ~~add brew recipe~~
