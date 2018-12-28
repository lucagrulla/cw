# cw

[![Release](https://img.shields.io/github/release/lucagrulla/cw.svg?style=flat-square)](https://github.com/lucagrulla/cw/releases/latest)
[![Software License](https://img.shields.io/badge/license-apache2-brightgreen.svg?style=flat-square)](LICENSE.md)
![Github All Releases](https://img.shields.io/github/downloads/lucagrulla/cw/total.svg)

The **best** way to tail AWS CloudWatch Logs from your terminal.

Author - [Luca Grulla](https://www.lucagrulla.com)  - [https://www.lucagrulla.com](https://www.lucagrulla.com)


* [Features](##features)
* [Installation](##installation)
* [Commands and options](#commands-and-options)
* [Examples](#examples)
* [AWS credentials and configuration](#AWS-credentials-and-configuration)
* [v2.* to v3.* command syntax breaking changes](#v2.*-to-v3.*-command-syntax-breaking-changes)

## Features

* **No external dependencies** (no pip, npm, rubygems) and easy installation.
  * cw is a native executable targeting your OS.
* **Fast**. cw is written in golang and compiled against your architecture. There is no intermediate runtime.
* **Flexible date and time parser**.
  * You can work with either `Local` timezone or `UTC` (default).
  * Flexible parsing.
    * Human friendly formats, i.e. `1h20m`  to indicate 1 hour and 20 minutes ago.
    * a specific hour, i.e. `13:10` to indicate 13:10 of today.
    * a full timestamp `2018-10-20T8:53`.
* Powerful built-in **grep** (`--grep`) and **grepv** (`--grepv`).
* Work smoothly with piping, i.e. `cw tail -f my-stream >> myfile.txt`.
* Coloured output (but use `--no-color` to disable if needed).
* Flexibile credentials control.
  * By default it uses the **AWS .aws/credentials and .aws/profile** files. Overrides can be done with the  `--profile` and `--region` flags.

## Installation

### On Mac OSX using [Homebrew](https://brew.sh)

```bash
brew tap lucagrulla/cw
brew install cw
```

### On Windows using Scoop.sh

```bash
scoop bucket add cw https://github.com/lucagrulla/cw-scoop-bucket.git
scoop install cw
```

### Using Go tools

```bash
go get github.com/lucagrulla/cw
```

## Commands and options

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
    -c, --no-color         Disable coloured output.
        --version          Show application version.

    Subcommands:
    ls groups
        Show all groups.

    ls streams <group>
        Show all streams in a given log group.
    ```
* `cw tail` tail a given log group/log stream
    ```bash
        usage: cw tail [<flags>] <groupName:logStreamPrefix...>...

        Tail log groups/streams.

        Flags:
            --help             Show context-sensitive help (also try --help-long and --help-man).
        -p, --profile=PROFILE  The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.
        -r, --region=REGION    The target AWS region. By default cw will use the default region defined in the .aws/credentials file.
        -c, --no-color         Disable coloured output.
            --version          Show application version.
        -f, --follow           Don't stop when the end of streams is reached, but rather wait for additional data to be appended.
        -t, --timestamp        Print the event timestamp.
        -i, --event-id         Print the event Id.
        -s, --stream-name      Print the log stream name this event belongs to.
        -n, --group-name       Print the log log group name this event belongs to.
        -b, --start="2018-12-25T09:34:45"
                                The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of hours and minutes prior to the present. Denote hours with
                                'h' and minutes with 'm' i.e. 80m, 4h30m. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format:
                                2017-02-27[T09[:00[:00]].
        -e, --end=""           The UTC end time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of hours and minutes prior to the present. Denote hours with
                                'h' and minutes with 'm' i.e. 80m, 4h30m.If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format:
                                2017-02-27[T09[:00[:00]].
        -l, --local            Treat date and time in Local timezone.
        -g, --grep=""          Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.
        -v, --grepv=""         Equivalent of grep --invert-match. Invert match pattern to filter logs by.

        Args:
        <groupName:logStreamPrefix...>
            The log group and stream name, with group:prefix syntax.Stream name can be just the prefix. If no stream name is specified all stream names in the given group will be tailed.Multiple group/stream
            tuple can be passed. e.g. cw tail group1:prefix group2:prefix group3:prefix.     
    ```

## Examples

* list of the available log groups
  * `cw ls groups`
* list of the log streams in a given log group
  * `cw ls streams my-log-group`
* tail and follow given log groups/streams
  * `cw tail -f my-log-group`
  * `cw tail -f my-log-group:my-log-stream-prefix`
  * `cw tail -f my-log-group:my-log-stream-prefix my-log-group2`
  * `cw tail -f my-log-group:my-log-stream-prefix -b2017-01-01T08:10:10 -e2017-01-01T08:05:00`  
  * `cw tail -f my-log-group:my-log-stream-prefix -b3h` to start from 3 hours ago.
  * `cw tail -f my-log-group:my-log-stream-prefix -b100m`  to start from 100 minutes ago.
  * `cw tail -f my-log-group:my-log-stream-prefix -b2h30m`  to start from 2 hours and 30 minutes ago.
  * `cw tail -f my-log-group -b9:00 -e9:01`

## Time and Dates

Time and dates are treated as UTC by default.
If you prefer to use Local zone just set the ```--local``` flag.

## AWS credentials and configuration

`cw` uses the default credentials profile (stored in ./aws/credentials) for authentication and shared config (.aws/config) for identifying the target AWS region. Both profile and region are overridable with the  `profile` and `region` global flags.

## v2.* to v3.* command syntax breaking changes 

In v3.x the syntax of the ```tail``` command has changed.

* The optional begin and end tail timestamps are now command flags, `--begin` and `--end`.
* Log group and optional stream changed from being positional arguments to be a single argument tuple with `:` as a separator: `mygroup:mystream`
* Example:
  * old syntax: `cw tail my-group my-stream 2018-12-21:09 2018-12-21:12`
  * new syntax: `cw tail -b2018-12-21:09 -e2018-12-21:12 mygroup:mystream`