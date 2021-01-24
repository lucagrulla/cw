# cw

[![Release](https://img.shields.io/github/release/lucagrulla/cw.svg?style=flat-square)](https://github.com/lucagrulla/cw/releases/latest)
[![Software License](https://img.shields.io/badge/license-apache2-brightgreen.svg?style=flat-square)](LICENSE.md)
![Github All Releases](https://img.shields.io/github/downloads/lucagrulla/cw/total.svg)
![CircleCI branch](https://img.shields.io/circleci/project/github/lucagrulla/cw/master.svg?label=CircleCI)

![cw - the best way to tail AWS CloudWatch Logs](https://github.com/lucagrulla/cw/raw/master/images/cw-logo1280x640.png)


The **best** way to tail AWS CloudWatch Logs from your terminal.

Author - [Luca Grulla](https://www.lucagrulla.com)  - [https://www.lucagrulla.com](https://www.lucagrulla.com)


* [Features](##features)
* [Installation](#installation)
* [Commands and options](#commands-and-options)
* [Examples](#examples)
* [AWS credentials and configuration](#AWS-credentials-and-configuration)
* [Miscellaneous](#miscellaneous)
* [Release notes](https://github.com/lucagrulla/cw/wiki/Release-notes)

## Features

* **No external dependencies** 
  * cw is a native executable targeting your OS. No pip, npm, rubygems.
* **Fast**. 
  * cw is written in golang and compiled against your architecture.
* **Flexible date and time parser**.
  * Work with either `Local` timezone or `UTC` (default).
  * Flexible parsing.
    * Human friendly formats, i.e. `2d1h20m` to indicate 2 days, 1 hour and 20 minutes ago.
    * a specific hour, i.e. `13:10` to indicate 13:10 of today.
    * a full timestamp `2018-10-20T8:53`.
* **Multi log groups tailing**
   * tail multiple log groups  in parallel: `cw tail my-auth-service my-web`.
* Powerful built-in **grep** (`--grep`) and **grepv** (`--grepv`).
* **Pipe operator |** supported  
   * `echo my-group | cw tail` and `cat groups.txt | cw tail`. 
* **Redirection operator >>** supported 
   * `cw tail -f my-stream >> myfile.txt`.
* Coloured output
   * `--no-color` flag to disable if needed.
* Flexible credentials control.
   * By default the **AWS .aws/credentials and .aws/profile** files are used. Overrides can be achieved with the  `--profile` and `--region` flags.

## Installation

### Mac OSX

#### using [Homebrew](https://brew.sh)

```bash
brew tap lucagrulla/tap
brew install cw
```

### Linux

#### using [Linuxbrew](https://linuxbrew.sh/brew/)

```bash
brew tap lucagrulla/tap
brew install cw
```

#### .deb/.rpm

Download the ```.deb``` or ```.rpm``` from the [releases page](https://github.com/lucagrulla/cw/releases/latest) and install with ````dpkg -i```` and ````rpm -i```` respectively.

#### using [Snapcraft.io](https://snapcraft.io)
*Note*: If you upgrade to 3.3.0 please note the new alias command.This is required to comply with snapcraft new release rules.
```bash
snap install cw-sh
sudo snap connect cw-sh:dot-aws-config-credentials
sudo snap alias cw-sh.cw cw
```

`cw` runs with strict confinement; the `dot-aws-config-credentials` interface connection is required to have access to `.aws/config` and `.aws/credentials` files

[![Get it from the Snap Store](https://snapcraft.io/static/images/badges/en/snap-store-white.svg)](https://snapcraft.io/cw-sh)

### On Windows

#### using [Scoop.sh](https://scoop.sh/)

```bash
scoop bucket add cw https://github.com/lucagrulla/cw-scoop-bucket.git
scoop install cw
```

### Go tools

```bash
go get github.com/lucagrulla/cw
```

## Commands and options

### Global flags

* `--profile=profile-name` Override the AWS profile used for connection.
* `--region=aws-region` Override the target AWS region.
* `--no-color`         Disable coloured output.
* `--endpooint`         The target AWS endpoint url. By default cw will use the default aws endpoints.

### Commands

* `cw ls` list all the log groups/log streams within a group
    ```console
    usage: cw ls <command> [<args> ...]

    Show an entity

    Flags:
        --help             Show context-sensitive help (also try --help-long and --help-man).
        --profile=PROFILE  The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.
        --region=REGION    The target AWS region.. By default cw will use the default region defined in the .aws/credentials file.
        --endpoint=ENDPOINT-URL The target AWS endpoint url. By default cw will use the default aws
                         endpoints.
    -c, --no-color         Disable coloured output.
        --version          Show application version.

    Subcommands:
    ls groups
        Show all groups.

    ls streams <group>
        Show all streams in a given log group.
    ```
* `cw tail` tail a given log group/log stream
    ```console
        usage: cw tail [<flags>] <groupName:logStreamPrefix...>...

        Tail log groups/streams.

        Flags:
            --help             Show context-sensitive help (also try --help-long and --help-man).
            --profile=PROFILE  The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.
            --region=REGION    The target AWS region. By default cw will use the default region defined in the .aws/credentials file.
            --no-color         Disable coloured output.
            --version          Show application version.
        -f, --follow           Don't stop when the end of streams is reached, but rather wait for additional data to be appended.
        -t, --timestamp        Print the event timestamp.
        -i, --event-id         Print the event Id.
        -s, --stream-name      Print the log stream name this event belongs to.
        -n, --group-name       Print the log log group name this event belongs to.
        -b, --start="2018-12-25T09:34:45"
                               The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of days, hours and minutes prior to the present. Denote days with
                               'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format:
                               2017-02-27[T09[:00[:00]].
        -e, --end=""           The UTC end time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of days, hours and minutes prior to the present. Denote days with
                               'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format:
                               2017-02-27[T09[:00[:00]].
        -l, --local            Treat date and time in Local timezone.
        -r --retry             Keep trying to open a log group/log stream if it is inaccessible.
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
  * `cw tail -f my-log-group:my-log-stream-prefix -b7d` to start from 7 days ago.
  * `cw tail -f my-log-group:my-log-stream-prefix -b3h` to start from 3 hours ago.
  * `cw tail -f my-log-group:my-log-stream-prefix -b100m`  to start from 100 minutes ago.
  * `cw tail -f my-log-group:my-log-stream-prefix -b2h30m`  to start from 2 hours and 30 minutes ago.
  * `cw tail -f my-log-group -b9:00 -e9:01`

## Time and Dates

Time and dates are treated as UTC by default.
If you prefer to use Local zone just set the ```--local``` flag.

## AWS credentials and configuration

`cw` uses the default credentials profile (stored in ./aws/credentials) for authentication and shared config (.aws/config) for identifying the target AWS region. Both profile and region are overridable with the  `profile` and `region` global flags.

### AWS SSO
As today (May 2020) AWS Go SDK is not supporting AWS SSO correctly. 
The best approach is to use one of these tools while the SDK is updated:
https://github.com/benkehoe/aws-sso-credential-process
https://github.com/victorskl/yawsso

## Miscellaneous
### CW doesn't work behind a proxy
Please use ```HTTP_PROXY``` environment variable as required by AWS cli:
https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-proxy.html

## Breaking changes notes
Read [here](https://github.com/lucagrulla/cw/wiki/Breaking-changes-notes)
