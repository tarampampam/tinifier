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

> TinyPNG uses intelligent lossy compression techniques to reduce the file size of your WEBP, AVIF, JPEG, and
> PNG files. By selectively decreasing the number of colors in an image, fewer bytes are needed to store the
> data. The effect is nearly imperceptible, but it significantly reduces file size.
>
> Images uploaded to TinyPNG are automatically removed after a maximum of 48 hours, and only you are allowed
> to download them.

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

### 📦 Debian/Ubuntu-based (.deb) systems

Execute the following commands in order:

```shell
# setup the repository automatically
curl -1sLf https://dl.cloudsmith.io/public/tarampampam/tinifier/setup.deb.sh | sudo -E bash

# install the package
sudo apt install tinifier
```

<details>
  <summary>Uninstalling</summary>

```shell
sudo apt remove tinifier
rm /etc/apt/sources.list.d/tarampampam-tinifier.list
```

</details>

### 📦 RedHat (.rpm) systems

```shell
# setup the repository automatically
curl -1sLf https://dl.cloudsmith.io/public/tarampampam/tinifier/setup.rpm.sh | sudo -E bash

# install the package
sudo dnf install tinifier # RedHat, CentOS, etc.
sudo yum install tinifier # Fedora, etc.
sudo zypper install tinifier # OpenSUSE, etc.
```

<details>
  <summary>Uninstalling</summary>

```shell
# RedHat, CentOS, Fedora, etc.
sudo dnf remove tinifier
rm /etc/yum.repos.d/tarampampam-tinifier.repo
rm /etc/yum.repos.d/tarampampam-tinifier-source.repo

# OpenSUSE, etc.
sudo zypper remove tinifier
zypper rr tarampampam-tinifier
zypper rr tarampampam-tinifier-source
```

</details>

### 📦 Alpine Linux

```shell
# bash is required for the setup script
sudo apk add --no-cache bash

# setup the repository automatically
curl -1sLf https://dl.cloudsmith.io/public/tarampampam/tinifier/setup.alpine.sh | sudo -E bash

# install the package
sudo apk add tinifier
```

<details>
  <summary>Uninstalling</summary>

```shell
sudo apk del tinifier
$EDITOR /etc/apk/repositories # remove the line with the repository
```

</details>

### 📦 AUR (Arch Linux)

There are three packages available in the AUR:

- Build from source: [tinifier](https://aur.archlinux.org/packages/tinifier)
- Precompiled: [tinifier-bin](https://aur.archlinux.org/packages/tinifier-bin)

```shell
pamac build tinifier
```

<details>
  <summary>Uninstalling</summary>

```shell
pacman -Rs tinifier
```

</details>

### 📦 Binary (Linux, macOS, Windows)

Download the latest binary for your architecture/OS from the [releases page][link_releases]. For example, to install
the latest version to the `/usr/local/bin` directory on an **amd64** system (e.g., Debian, Ubuntu), you can run:

```shell
# download and install the binary
curl -SsL \
  https://github.com/tarampampam/tinifier/releases/latest/download/tinifier-linux-amd64.gz | \
  gunzip -c | sudo tee /usr/local/bin/tinifier > /dev/null

# make the binary executable
sudo chmod +x /usr/local/bin/tinifier
```

<details>
  <summary>Uninstalling</summary>

```shell
sudo rm /usr/local/bin/tinifier
```

</details>

> [!TIP]
> Each release includes binaries for **linux**, **darwin** (macOS) and **windows** (`amd64` and `arm64` architectures).
> You can download the binary for your system from the [releases page][link_releases] (section `Assets`). And - yes,
> all what you need is just download and run single binary file.

[link_releases]:https://github.com/tarampampam/tinifier/releases

### 📦 Docker image

Also, you can use the Docker image:

| Registry                               | Image                          |
|----------------------------------------|--------------------------------|
| [GitHub Container Registry][link_ghcr] | `ghcr.io/tarampampam/tinifier` |
| [Docker Hub][link_docker_hub] (mirror) | `tarampampam/tinifier`         |

> [!NOTE]
> It’s recommended to avoid using the `latest` tag, as **major** upgrades may include breaking changes.
> Instead, use specific tags in `:X.Y.Z` or only `:X` format for version consistency.

[link_ghcr]:https://github.com/tarampampam/tinifier/pkgs/container/tinifier
[link_docker_hub]:https://hub.docker.com/r/tarampampam/tinifier/

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
   CLI tool for compressing images using the TinyPNG.

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
   --keep-original-file         Leave the original (uncompressed) file next to the compressed one (with the .orig extension) [$KEEP_ORIGINAL_FILE]
   --help, -h                   Show help
   --version, -v                Print the version
```
<!--/GENERATED:APP_README-->

## 📜 License

This is open-sourced software licensed under the [MIT License][link_license].

[link_license]:https://github.com/tarampampam/tinifier/blob/master/LICENSE
