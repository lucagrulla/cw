# cw 
[![Release](https://img.shields.io/github/release/lucagrulla/cw.svg?style=flat-square)](https://github.com/lucagrulla/cw/releases/latest)
[![Software License](https://img.shields.io/badge/license-apache2-brightgreen.svg?style=flat-square)](LICENSE.md)


The **best** way to tail AWS Cloudwatch Logs from your terminal.

Author - [Luca Grulla](https://www.lucagrulla.com)  - [https://www.lucagrulla.com](https://www.lucagrulla.com)

## Commands and flags

### Global flags
* `-p`, `--profile=profile-name` Override the AWS profile used for connection
* `-r`, `--region=aws-region` Override the target AWS region 

### Commands

* `cw ls` list all the log groups/log streams within a group
* `cw tail` tail a given log group/log stream
  * flags
    *  `-f`, `--follow`       Don't stop when the end of stream is reached, but rather wait for additional data to be appended.
    *  `-t`, `--timestamp`    Print the event timestamp.
    *  `-i`, `--event Id`     Print the event Id.
    *  `-s`, `--stream name`  Print the log stream name this event belongs to.
    *  `-g`, `--grep=""`      Pattern to filter logs by.
    *  `-v`, `--grepv=""`     Invert-match logs filter (as per grep -v).

## Examples

* list of the available log groups
  * `cw ls groups`
* list of the log streams in a given log group
  * `cw ls streams my-log-group`
* tail and follow a given log group/stream
  * `cw tail -f my-log-group` 
  * `cw tail -f my-log-group my-log-stream-prefix` 
  * `cw tail -f my-log-group my-log-stream-prefix 2017-01-01T08:10:10 2017-01-01T08:05:00`  
  * `cw tail -f my-log-group \* 9:00 9:01` The use of the \* wildchar will let you tail all the log streams in my-log-group. 

Time and dates are always treated in UTC.


## AWS credentials and configuration

`cw` uses the default credentials profile (stored in ./aws/credentials) for authentication and shared config (.aws/config) for identifying the target AWS region. Both profile and region are overridable with the  `profile` and `region` global flags.

 
## Installation

On Mac OSX:

* `brew tap lucagrulla/cw`
* `brew install cw`

Using go tools:

`go get github.com/lucagrulla/cw`
