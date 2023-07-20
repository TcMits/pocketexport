# pocketexport

The PocketExport plugin is a Go-based plugin for PocketBase. This plugin enables the creation and management of export requests within PocketBase. It provides functionality to store and retrieve information related to export requests, such as filters, sorting preferences, target collection, and file format.

## usage

### register

```go
package main

import (
  "log"

  _ "github.com/TcMits/pocketexport/migrations"
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

### apis

to create an export
```js
import PocketBase from 'pocketbase';

const pb = new PocketBase('http://0.0.0.0:8090');

// example create data
const data = {
    "exportCollectionName": "users", // export 'users' collection
    "headers": [
        {
            "fieldName": "name",
            "header": "Tên"
        },
        {
            "fieldName": "updated",
            "header": "Ngày cập nhật",
            "timezone": "Asia/Ho_Chi_Minh"
        },
        {
            "fieldName": "gender",
            "header": "Lựa chọn",
            "valueMap": {
                "male": "Nam",
                "female": "Nữ"
            }
        }
    ],
    "filter": "name != \"\"",
    "sort": "name",
    "format": "csv",
    "ownerId": "4gw3vii9aopnnvc",
    "ownerCollectionName": "" // empty str means that you are admin
};

const record = await pb.collection('pocketexport_exports').create(data);
```
