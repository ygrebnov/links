[![Links](https://yaroslavgrebnov.com/assets/images/links_go_tool_for_checking_the_status_of_page_links-a0facf7d8d4e437a00dd689a2a462d58.png)](https://yaroslavgrebnov.com/projects/links/overview)

**Links** is a fast and configurable Go tool for checking the status of page links on a given host. Written by [ygrebnov](https://github.com/ygrebnov).

---

[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/links)](https://pkg.go.dev/github.com/ygrebnov/links)
[![Build Status](https://github.com/ygrebnov/links/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/links/actions/workflows/build.yml)
[![codecov]([![codecov](https://codecov.io/gh/ygrebnov/links/graph/badge.svg?token=7SY34YUHRW)](https://codecov.io/gh/ygrebnov/links))](https://codecov.io/gh/ygrebnov/links)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/links)](https://goreportcard.com/report/github.com/ygrebnov/links)

[User Guide](https://yaroslavgrebnov.com/projects/links/overview) | [Commands](https://yaroslavgrebnov.com/projects/links/usage) | [Configuration](https://yaroslavgrebnov.com/projects/links/configuration) | [Contributing](#contributing)

## Features

- Inspect internal and external links with flexible configuration options.
- Supports multiple output formats: stdout, HTML, and CSV.
- Supports detailed configuration of pages inspecting and results outputting.

## Installation

**homebrew tap**
```shell
brew install ygrebnov/tap/links
```

**go install**
```shell
go install github.com/ygrebnov/links
```

**manually**

Archives with pre-compiled binaries can be downloaded from [releases page](https://github.com/ygrebnov/links/releases). 

## Usage

The most basic example. Inspects links at `http://example.com` starting from `/` and outputs results to stdout:

```shell
links inspect --host=example.com
```

Start from some other page:

```shell
links inspect --host=example.com --start=/some-page
```

Output results to an HTML file and open it in the default browser:

```shell
links inspect --host=example.com -o html
```

Do not output results for links returning 200 status codes:

```shell
links inspect --host=example.com --skipok
```

## Configuration

There are several ways to configure the tool. The configuration can be set using command line options, a dedicated command, environment variables, or a configuration file. See [User Guide Configuration Section](https://yaroslavgrebnov.com/projects/links/configuration) for more details.

Display current configuration:

```shell
links config show
```

Use a custom configuration file:

```shell
links inspect --host=example.com --config /path/to/config.yaml
```

Configuration file example:
```yaml
inspector:
    requestTimeout: 30s
    doNotFollowRedirects: false
    logExternalLinks: false
    skipStatusCodes:
        - 200
        - 301
        - 302
    retryAttempts: 3
    retryDelay: 2ms
printer:
    sortOutput: false
    displayOccurrences: false
    skipOK: false
    doNotOpenFileReport: false
```

## Output formats

With `printer.sortOutput` = `false`, `printer.displayOccurrences` = `false`, and `printer.outputFormat` = `stdout`, results are printed out on-the-fly.

Generated HTML report is opened in the default browser. CSV file is opened in the application associated with .csv files. This behavior can be changed by setting `printer.doNotOpenFileReport` to `true`.

Columns in an HTML report can be sorted by clicking on the column header.

## Contributing

Contributions are welcome!  
Please open an [issue](https://github.com/ygrebnov/links/issues) or submit a [pull request](https://github.com/ygrebnov/links/pulls).

## License

This project is licensed under Apache License 2.0, see the [LICENSE](LICENSE) file for details.