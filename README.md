# Github installer for Plugin Registry

[![GitHub Releases](https://img.shields.io/github/v/release/nhatthm/plugin-registry-github)](https://github.com/nhatthm/plugin-registry-github/releases/latest)
[![Build Status](https://github.com/nhatthm/plugin-registry-github/actions/workflows/test.yaml/badge.svg)](https://github.com/nhatthm/plugin-registry-github/actions/workflows/test.yaml)
[![codecov](https://codecov.io/gh/nhatthm/plugin-registry-github/branch/master/graph/badge.svg?token=eTdAgDE2vR)](https://codecov.io/gh/nhatthm/plugin-registry-github)
[![Go Report Card](https://goreportcard.com/badge/github.com/nhatthm/plugin-registry-github)](https://goreportcard.com/report/github.com/nhatthm/plugin-registry-github)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/nhatthm/plugin-registry-github)
[![Donate](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://www.paypal.com/donate/?hosted_button_id=PJZSGJN57TDJY)

An installer for [plugin-registry](https://github.com/nhatthm/plugin-registry)

## Prerequisites

- `Go >= 1.15`

## Install

```bash
go get github.com/nhatthm/plugin-registry-github
```

## Usage

Import the library while bootstrapping the application (see the [examples](#examples))

The installer supports this source format: `[https?://]github.com/owner/repository[@version]`. For examples:
- `https://github.com/owner/repository`
- `github.com/owner/repository@latest`
- `github.com/owner/repository@v1.4.2`

In the root folder of the repository, there must be a `.plugin.registry.yaml` file that describe the plugin. 
For example: https://github.com/nhatthm/moneylovercli-plugin-n26/blob/master/.plugin.registry.yaml

## Examples

```go
package mypackage

import (
	"context"

	registry "github.com/nhatthm/plugin-registry"
	_ "github.com/nhatthm/plugin-registry-github" // Add file system installer.
)

var defaultRegistry = mustCreateRegistry()

func mustCreateRegistry() registry.Registry {
	r, err := createRegistry()
	if err != nil {
		panic(err)
	}

	return r
}

func createRegistry() (registry.Registry, error) {
	return registry.NewRegistry("~/plugins")
}

func installPlugin(url string) error {
	return defaultRegistry.Install(context.Background(), url)
}

```

## Donation

If this project help you reduce time to develop, you can give me a cup of coffee :)

### Paypal donation

[![paypal](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/donate/?hosted_button_id=PJZSGJN57TDJY)

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;or scan this

<img src="https://user-images.githubusercontent.com/1154587/113494222-ad8cb200-94e6-11eb-9ef3-eb883ada222a.png" width="147px" />
