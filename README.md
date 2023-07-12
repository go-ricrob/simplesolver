# go-ricrob

[![Go Report Card](https://goreportcard.com/badge/github.com/stfnmllr/go-ricrob)](https://goreportcard.com/report/github.com/stfnmllr/go-ricrob)
[![REUSE status](https://api.reuse.software/badge/git.fsfe.org/reuse/api)](https://api.reuse.software/info/git.fsfe.org/reuse/api)
![](https://github.com/stfnmllr/go-ricrob/workflows/build/badge.svg)

ricochet robots

## Building

To build go-ricrob you need to have a working Go environment of the [latest or second latest Go version](https://golang.org/dl/).

## Test

### Test server HTTP API

```
#Assuming the server is running at 'localhost:5000' the following command
#retrieves a board definition via tile position URL query parameters.
curl "http://localhost:50000/board?ttl=A1F&ttr=A2F&tbl=A3F&tbr=A4F"
```

## ASCII art 
ASCII Art generated at https://patorjk.com/software/taag/.