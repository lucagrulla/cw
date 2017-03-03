# cw

A CLI tool for easier interaction with AWS Cloudwatch.

It provides commands for:

* tail a given log group/stream
* list of the available log groups

`cw` uses the default credentials profile(stored in .aws) to authenticate against AWS.
 
## Installation

On Mac:

* `brew tap lucagrulla/cw`
* `brew install cw`

## TODOs:

* ~~fix bug for long polling once events are finished(currently we print again a last chunk of alerts)~~
* ~~add an optionl end date for time window~~
* allow more flexible startTime format(no seconds means 00, no minutes means 00:00)
* add coloured output
* ~~add brew recipe~~
