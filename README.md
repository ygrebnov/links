# Links

A Go-based tool for checking pages links statuses at the given host.

Author: Yaroslav Grebnov, [grebnov@gmail.com](mailto:grebnov@gmail.com).

## Features

- Inspect links with customizable configurations.
- Supports multiple output formats: stdout, HTML, and CSV.
- Supports detailed configuration of pages inspecting and results outputting.

## Installation

**homebrew tap**
```shell
brew install ygrebnov/tap/links
```

**homebrew**
```shell
brew tap ygrebnov/tap
brew install links
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

Output results to HTML file and open it in the default browser:

```shell
links inspect --host=example.com -o html
```

Do not output results for links returning 200 status code:

```shell
links inspect --host=example.com --skipok
```

## Configuration

Display current configuration:

```shell
links config show
```

Change configuration using command line. Note that `config set` command results in the configuration being saved to the file. The file is created if it does not exist.

```shell
links config set printer.skipOK true
```

Alternatively, configuration can be set via environment variables. The environment variable name is a combination of the command line option name and the `LINKS_` prefix. For example, `printer` `skipOK` setting can be configured using environment variable `LINKS_PRINTER_SKIPOK`.

```shell
LINKS_PRINTER_SKIPOK=true links inspect --host=example.com
```

## Configuration options description
| Option | Description                                                                 | Default value | Environment variable name |
|--------|-----------------------------------------------------------------------------|---------------|---------------------------|
| inspector.requestTimeout | HTTP request timeout                                                        | 30s           | `LINKS_INSPECTOR_REQUESTTIMEOUT`        |
| inspector.doNotFollowRedirects | Do not follow redirects                                                     | false         | `LINKS_INSPECTOR_DONOTFOLLOWREDIRECTS` |
| inspector.logExternalLinks | Log external links. Links with other hostname are considered to be external | false         | `LINKS_INSPECTOR_LOGEXTERNALLINKS` |
| inspector.skipStatusCodes | Do not output results for links with the specified status codes             | empty list    | `LINKS_INSPECTOR_SKIPSTATUSCODES`       |
| inspector.retryAttempts | Number of retry attempts for failed requests                                | 3             | `LINKS_INSPECTOR_RETRYATTEMPTS`         |
| inspector.retryDelay | Delay between retry attempts                                                | 2ms           | `LINKS_INSPECTOR_RETRYDELAY`           |
| printer.sortOutput | Sort output by link URL                                                     | false         | `LINKS_PRINTER_SORTOUTPUT`             |
| printer.displayOccurrences | Display number of occurrences for each link                                 | false         | `LINKS_PRINTER_DISPLAYOCCURRENCES`     |
| printer.skipOK | Do not output results for links returning 200 status code                   | false         | `LINKS_PRINTER_SKIPOK`                 |
| printer.outputFormat | Results output format. Supported formats: `stdout`, `html`, `csv`           | stdout        | `LINKS_PRINTER_OUTPUTFORMAT`           |
| printer.doNotOpenFileReport | Do not open file report in the default browser                              | false         | `LINKS_PRINTER_DONOTOPENFILEREPORT`    |

## Configuration file

One more option is to set configuration using the configuration file in `yaml` format. By default, tool tries to load file from the default location, which is different on each operating system. Exact path is displayed on the first line of `links config show` command output.

The configuration file location can be overridden by specifying the `--config` option.

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

Columns in HTML report can be sorted by clicking on the column header.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.  

## License

This project is licensed under Apache License 2.0 - see the [LICENSE](LICENSE) file for details.