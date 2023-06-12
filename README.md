# pocketexport

The PocketExport plugin is a Go-based plugin for PocketBase. This plugin enables the creation and management of export requests within PocketBase. It provides functionality to store and retrieve information related to export requests, such as filters, sorting preferences, target collection, and file format.

## Usage

```go
package main

import (
  "log"

  "github.com/TcMits/pocketexport"
  "github.com/pocketbase/pocketbase"
)

func main() {
  app := pocketbase.New()

  // register pocketexport app
  if err := pocketexport.Register(app); err != nil {
    log.Fatal(err)
  }

  if err := app.Start(); err != nil {
    log.Fatal(err)
  }
}
```
