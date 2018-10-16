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
    ```bash
    usage: cw ls <command> [<args> ...]

    Show an entity

    Flags:
        --help             Show context-sensitive help (also try --help-long and --help-man).
    -p, --profile=PROFILE  The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.
    -r, --region=REGION    The target AWS region.. By default cw will use the default region defined in the .aws/credentials file.
        --version          Show application version.

    Subcommands:
    ls groups
        Show all groups.

    ls streams <group>
        Show all streams in a given log group.
    ```
* `cw tail` tail a given log group/log stream
    ```bash
        usage: cw tail [<flags>] <group> [<stream>] [<start>] [<end>]

        Tail a log group.

        Flags:
            --help             Show context-sensitive help (also try --help-long and --help-man).
        -p, --profile=PROFILE  The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.
        -r, --region=REGION    The target AWS region.. By default cw will use the default region defined in the .aws/credentials file.
            --version          Show application version.
        -f, --follow           Don't stop when the end of stream is reached, but rather wait for additional data to be appended.
        -t, --timestamp        Print the event timestamp.
        -i, --event Id         Print the event Id
        -s, --stream name      Print the log stream name this event belongs to.
        -g, --grep=""          Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.
        -v, --grepv=""         equivalent of grep --invert-match. Invert match pattern to filter logs by.

        Args:
        <group>     The log group name.
        [<stream>]  The log stream name. Use \* for tail all the group streams.
        [<start>]   The start time. Passed as either UTC or human-friendly format. The human-friendly version accepts a number of hours and mninutes ago from now. Use 'h' to identify                 hours. 'm' to identify minutes. i.e. 4h30m If a timestamp is passed (format: hh[:mm]) it is expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].
        [<end>]     The end time. Passed as either UTC or human-friendly format. The human-friendly version accepts a number of hours and mninutes ago from now. Use 'h' to identify                    hours. 'm' to identify minutes. i.e. 4h30m. If a timestamp is passed (format: hh[:mm]) it is expanded to today at the given time. Full format: 2017-02-27[T09:00[:00].
    ```

## Examples

* list of the available log groups
  * `cw ls groups`
* list of the log streams in a given log group
  * `cw ls streams my-log-group`
* tail and follow a given log group/stream
  * `cw tail -f my-log-group`
  * `cw tail -f my-log-group my-log-stream-prefix`
  * `cw tail -f my-log-group my-log-stream-prefix 2017-01-01T08:10:10 2017-01-01T08:05:00`  
  * `cw tail -f my-log-group my-log-stream-prefix 3h` to start from 3 hours ago.
  * `cw tail -f my-log-group my-log-stream-prefix 100m`  to start from 100 minutes ago.
  * `cw tail -f my-log-group my-log-stream-prefix 2h:30m`  to start from 2 hours and 30 minutes ago.
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
