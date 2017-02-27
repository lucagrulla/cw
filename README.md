# cw

A CLI tool to interact with AWS Cloudwatch.

It provides commands for:

* tail a log group/stream
* list of the available log groups


cw uses the .aws/ default credentials profile to authenticate agansint AWS.
 
##TODOs:
** fix bug for long polling once events are finished(currently we print again a last chunk of alerts)
** allow more flexible startTime format(no seconds means 00, no minutes means 00:00)
** add coloured output
