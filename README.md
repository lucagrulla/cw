# cw

[![Release](https://img.shields.io/github/release/lucagrulla/cw.svg?style=flat-square)](https://github.com/lucagrulla/cw/releases/latest)
[![Software License](https://img.shields.io/badge/license-apache2-brightgreen.svg?style=flat-square)](LICENSE.md)
![Github All Releases](https://img.shields.io/github/downloads/lucagrulla/cw/total.svg)
![CircleCI branch](https://img.shields.io/circleci/project/github/lucagrulla/cw/master.svg?label=CircleCI)

![cw - the best way to tail AWS CloudWatch Logs](https://github.com/lucagrulla/cw/raw/master/images/cw-logo1280x640.png)

The **best** way to tail AWS CloudWatch Logs from your terminal.

Author - [Luca Grulla](https://www.lucagrulla.com) - [https://www.lucagrulla.com](https://www.lucagrulla.com)


* [Features](##features)
* [Installation](#installation)
* [Commands and options](#commands-and-options)
* [Examples](#examples)
* [AWS credentials and configuration](#AWS-credentials-and-configuration)
* [Miscellaneous](#miscellaneous)
* [Release notes](https://github.com/lucagrulla/cw/wiki/Release-notes)

## Features

-   **No external dependencies**
    -   cw is a native executable targeting your OS. No pip, npm, rubygems.
-   **Fast**.
    -   cw is written in golang and compiled against your architecture.
-   **Flexible date and time parser**.
    -   Work with either `Local` timezone or `UTC` (default).
    -   Flexible parsing.
        -   Human friendly formats, i.e. `2d1h20m` to indicate 2 days, 1 hour and 20 minutes ago.
        -   a specific hour, i.e. `13:10` to indicate 13:10 of today.
        -   a full timestamp `2018-10-20T8:53`.
-   **Multi log groups tailing**
    -   tail multiple log groups in parallel: `cw tail my-auth-service my-web`.
-   Powerful built-in **grep** (`--grep`) and **grepv** (`--grepv`).
-   [JMESPath](https://jmespath.org/) support for JSON queries (matching the [AWS CLI `--query`](https://docs.aws.amazon.com/cli/latest/userguide/cli-usage-filter.html#cli-usage-filter-client-side) flag)
-   **Pipe operator** supported
    -   `echo my-group | cw tail` and `cat groups.txt | cw tail`.
-   **Redirection operator >>** supported
    -   `cw tail -f my-stream >> myfile.txt`.
-   Coloured output
    -   `--no-color` flag to disable if needed.
-   Flexible credentials control.
    -   By default the **AWS .aws/credentials and .aws/profile** files are used. Overrides can be achieved with the `--profile` and `--region` flags.

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

Download the `.deb` or `.rpm` from the [releases page](https://github.com/lucagrulla/cw/releases/latest) and install with `dpkg -i` and `rpm -i` respectively.

#### using [Snapcraft.io](https://snapcraft.io)

_Note_: If you upgrade to 3.3.0 please note the new alias command. This is required to comply with snapcraft new release rules.

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

-   `--profile=profile-name` Override the AWS profile used for connection.
-   `--region=aws-region` Override the target AWS region.
-   `--no-color` Disable coloured output.
-   `--endpoint` The target AWS endpoint url. By default cw will use the default aws endpoints.
-   `--no-version-check` Ignore checks if a newer version of the module is available.

### Commands

-   `cw ls` list all the log groups/log streams within a group

    ```console
    Usage: cw ls <command>

    show an entity

    Flags:
      -h, --help               Show context-sensitive help.
          --endpoint=URL       The target AWS endpoint url. By default cw will use the default aws endpoints. NOTE: v4.0.0
                              dropped the flag short version.
          --profile=PROFILE    The target AWS profile. By default cw will use the default profile defined in the
                              .aws/credentials file. NOTE: v4.0.0 dropped the flag short version.
          --region=REGION      The target AWS region. By default cw will use the default region defined in the
                              .aws/credentials file. NOTE: v4.0.0 dropped the flag short version.
          --no-color           Disable coloured output.NOTE: v4.0.0 dropped the flag short version.
          --version            Print version information and quit
          --no-version-check   Ignore checks if a newer version of the module is available.

    Commands:
      ls groups
        Show all groups.

      ls streams <group>
        Show all streams in a given log group.

    cw: error: expected one of "groups",  "streams"
    ```

-   `cw tail` tail a given log group/log stream

    ```console
    Usage: cw tail <groupName[:logStreamPrefix]> ...

    Tail log groups/streams.

    Arguments:
      <groupName[:logStreamPrefix]> ...    The log group and stream name, with group:prefix syntax. Stream name can be just the prefix. If no stream name is specified all stream names in the given
                                          group will be tailed. Multiple group/stream tuple can be passed. e.g. cw tail group1:prefix1 group2:prefix2 group3:prefix3.

    Flags:
      -h, --help                           Show context-sensitive help.
          --endpoint=URL                   The target AWS endpoint url. By default cw will use the default aws endpoints. NOTE: v4.0.0 dropped the flag short version.
          --profile=PROFILE                The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file. NOTE: v4.0.0 dropped the flag short version.
          --region=REGION                  The target AWS region. By default cw will use the default region defined in the .aws/credentials file. NOTE: v4.0.0 dropped the flag short version.
          --no-color                       Disable coloured output.NOTE: v4.0.0 dropped the flag short version.
          --version                        Print version information and quit
          --no-version-check               Ignore checks if a newer version of the module is available.

      -f, --follow                         Don't stop when the end of streams is reached, but rather wait for additional data to be appended.
      -t, --timestamp                      Print the event timestamp.
      -i, --event-id                       Print the event Id.
      -s, --stream-name                    Print the log stream name this event belongs to.
      -n, --group-name                     Print the log group name this event belongs to.
      -r, --retry                          Keep trying to open a log group/log stream if it is inaccessible.
      -b, --start="2021-04-11T08:21:52"    The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of days, hours and minutes prior to
                                          the present. Denote days with 'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h. If just time is used (format: hh[:mm]) it is expanded to
                                          today at the given time. Full available date/time format: 2017-02-27[T09[:00[:00]].
      -e, --end=STRING                     The UTC end time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of days, hours and minutes prior to the
                                          present. Denote days with 'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h. If just time is used (format: hh[:mm]) it is expanded to today at
                                          the given time. Full available date/time format: 2017-02-27[T09[:00[:00]].
      -l, --local                          Treat date and time in Local timezone.
      -g, --grep=STRING                    Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.
      -v, --grepv=STRING                   Equivalent of grep --invert-match. Invert match pattern to filter logs by.
      -q, --query=STRING                   Equivalent of the --query flag in AWS CLI. Takes a JMESPath expression to filter JSON logs by. If the query fails (e.g. the log message was not JSON) then the original line is returned.
    ```

## Examples

-   list of the available log groups
    -   `cw ls groups`
-   list of the log streams in a given log group
    -   `cw ls streams my-log-group`
-   tail and follow given log groups/streams

    -   `cw tail -f my-log-group`
    -   `cw tail -f my-log-group:my-log-stream-prefix`
    -   `cw tail -f my-log-group:my-log-stream-prefix my-log-group2`
    -   `cw tail -f my-log-group:my-log-stream-prefix -b2017-01-01T08:10:10 -e2017-01-01T08:05:00`
    -   `cw tail -f my-log-group:my-log-stream-prefix -b7d` to start from 7 days ago.
    -   `cw tail -f my-log-group:my-log-stream-prefix -b3h` to start from 3 hours ago.
    -   `cw tail -f my-log-group:my-log-stream-prefix -b100m` to start from 100 minutes ago.
    -   `cw tail -f my-log-group:my-log-stream-prefix -b2h30m` to start from 2 hours and 30 minutes ago.
    -   `cw tail -f my-log-group -b9:00 -e9:01`

-   query JSON logs using [JMESPath](https://jmespath.org/) syntax
    -   `cw tail -f my-log-group --query "machines[?state=='running'].name"`

## Time and Dates

Time and dates are treated as UTC by default.
Use the `--local` flag if you prefer to use Local zone.

## AWS credentials and configuration

`cw` uses the default credentials profile (stored in ./aws/credentials) for authentication and shared config (.aws/config) for identifying the target AWS region. Both profile and region are overridable via the `profile` and `region` global flags.

### AWS SSO

AWS SSO is supported if you:

* use a CLI profile (either `default` or an alternate named profile) that includes the various SSO properties
  * `sso_start_url`, `sso_account_id`, `sso_role_name`, etc
* have a valid, active SSO session
  * via `aws sso login`

If you get an error message that includes `...failed to sign request: failed to retrieve credentials: the SSO session has expired or is invalid...` then you should renew your SSO session via `aws sso login` (and specify the named profile, if appropriate).

## Miscellaneous

### Use `cw` behind a proxy

Please use `HTTP_PROXY` environment variable as required by AWS cli:
<https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-proxy.html>

## Breaking changes notes

Read [here](https://github.com/lucagrulla/cw/wiki/Breaking-changes-notes)
