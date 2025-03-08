<p align="center">
  <a href="https://github.com/tarampampam/tinifier#readme">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://socialify.git.ci/tarampampam/tinifier/image?description=1&font=Raleway&forks=1&issues=1&logo=https%3A%2F%2Ftinypng.com%2Fimages%2Fapng%2Fpanda-waving.png&owner=1&pulls=1&pattern=Solid&stargazers=1&theme=Dark">
      <img align="center" src="https://socialify.git.ci/tarampampam/tinifier/image?description=1&font=Raleway&forks=1&issues=1&logo=https%3A%2F%2Ftinypng.com%2Fimages%2Fapng%2Fpanda-waving.png&owner=1&pulls=1&pattern=Solid&stargazers=1&theme=Light">
    </picture>
  </a>
</p>

# Tinifier

`tinifier` is a CLI tool for compressing images using the [TinyPNG](https://tinypng.com) API, with parallel
processing to speed up the workflow.

![demo](art/demo.gif)

## 🔥 Features list

- Compress images in **parallel** (configurable number of threads)
- Support for **multiple API keys** (automatically switches if a key exceeds its quota)
- Automatic **retries** for failed operations
- **Recursive search** for images in directories (configurable file extensions)
- Skip files if the difference between the original and compressed file sizes is below a specified percentage
- **Preserve the original file modification date/time** (including EXIF metadata), ensuring correct photo
  ordering (e.g., from smartphones) after compression

## 🧩 Installation

> TODO

## ⚙ Configuration

You can configure `tinifier` using a YAML file. Refer to [this example](tinifier.example.yml) for
available options.

You can specify the configuration file's location using the `--config-file` option. By default, however, the
tool searches for the file in the user's configuration directory:

- **Linux**: `~/.configs/tinifier.yml`
- **Windows**: `%APPDATA%\tinifier.yml`
- **macOS**: `~/Library/Application Support/tinifier.yml`

## 🚀 Use Cases (usage examples)

> [!IMPORTANT]
> A [TinyPNG](https://tinypng.com) API key is required. To obtain one:
> - Visit [tinypng.com/developers](https://tinypng.com/developers)
> - Fill out the form (enter your name and email address) and click "Get your API key"
> - Check your email and click the verification link
> - Activate your API key on the dashboard and save it

> [!TIP]
> If you need to process a large number of files and have a Gmail account, you can use the following
> trick - register multiple accounts on tinypng.com using aliases such as `your_mailbox+key1@gmail.com`,
> `your_mailbox+key2@gmail.com`, etc. This allows you to use a single mailbox to retrieve as many free API
> keys as needed.

#### ☝ Compress a Single Image

```shell
tinifier -k 'YOUR-API-KEY-GOES-HERE' ./img.png
```

#### ☝ Compress All PNG Images in a Directory and Two Other Images

```shell
tinifier -k 'API-KEY-1,API-KEY-2' -e png ./images-directory ./img-1.png ./img-2.png
```

#### ☝ Compress JPG and PNG Images in a Directory (Recursively) Using 20 Threads

```shell
tinifier -k 'YOUR-API-KEY-GOES-HERE' --ext png,jpg --threads 20 -r ./some-dir
```

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

## 📜 License

This is open-sourced software licensed under the [MIT License][link_license].

[link_license]:https://github.com/tarampampam/tinifier/blob/master/LICENSE
