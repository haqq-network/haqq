<!--
order: 1
-->

# Installation

Build and install the Haqq binaries from source or using Docker. {synopsis}

## Pre-requisites

- [Install Go 1.18+](https://golang.org/dl/) {prereq}
- [Install jq](https://stedolan.github.io/jq/download/) {prereq}

## Install Go

::: warning
Haqq is built using [Go](https://golang.org/dl/) version `1.18+`
:::

```bash
go version
```

:::tip
If the `haqqd: command not found` error message is returned, confirm that your [`GOPATH`](https://golang.org/doc/gopath_code#GOPATH) is correctly configured by running the following command:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

:::

## Install Binaries

::: tip
The latest {{ $themeConfig.project.name }} [version](https://github.com/haqq-network/haqq/releases) is `{{ $themeConfig.project.binary }} {{ $themeConfig.project.latest_version }}`
:::

### GitHub

Clone and build {{ $themeConfig.project.name }} using `git`:

```bash
git clone https://github.com/haqq-network/haqq.git
cd haqq
make install
```

Check that the `{{ $themeConfig.project.binary }}` binaries have been successfully installed:

```bash
haqqd version
```

### Docker

**🚧 `In developing...` 🏗️**

### Releases

You can also download a specific release available on the {{ $themeConfig.project.name }} [repository](https://github.com/haqq-network/haqq/releases) or via command line:

```bash
go install github.com/haqq-network/haqq@latest
```
