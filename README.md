
<!--GENERATED:APP_README-->
## 💻 Command line interface

```
Description:
   CLI client for images compressing using tinypng.com API.

Usage:
   tinifier [<options>] [<files-or-directories>]

Version:
   0.0.0@undefined

Options:
   --config-file="…", -c="…"    Path to the configuration file (default: depends/on/your-os/tinifier.yml) [$CONFIG_FILE]
   --api-key="…", -k="…"        TinyPNG API keys <https://tinypng.com/dashboard/api> (separated by commas) [$API_KEYS]
   --ext="…", -e="…"            Extensions of files to compress (separated by commas) (default: png,jpeg,jpg,webp,avif) [$FILE_EXTENSIONS]
   --threads="…", -t="…"        Number of threads to use for compressing (default: 16) [$THREADS]
   --max-errors="…"             Maximum number of errors to stop the process (set 0 to disable) (default: 10) [$MAX_ERRORS]
   --retry-attempts="…"         Number of retry attempts for upload/download/replace operations (default: 3) [$RETRY_ATTEMPTS]
   --delay-between-retries="…"  Delay between retry attempts (default: 1s) [$DELAY_BETWEEN_RETRIES]
   --recursive, -r              Search for files in listed directories recursively [$RECURSIVE]
   --skip-if-diff-less="…"      Skip files if the diff between the original and compressed file sizes < N% (default: 1) [$SKIP_IF_DIFF_LESS]
   --preserve-time, -p          Preserve the original file modification date/time (including EXIF) [$PRESERVE_TIME]
   --help, -h                   Show help
   --version, -v                Print the version
```
<!--/GENERATED:APP_README-->
