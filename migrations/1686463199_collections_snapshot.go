package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `[
			{
				"id": "utge0b58a4971cg",
				"created": "2023-06-11 05:57:46.552Z",
				"updated": "2023-06-11 05:57:46.552Z",
				"name": "pocketexport_exports",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "bpyvm35c",
						"name": "exportCollectionName",
						"type": "text",
						"required": true,
						"unique": false,
						"options": {
							"min": 1,
							"max": 256,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "uw6wisaz",
						"name": "headers",
						"type": "json",
						"required": true,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "mrazbt2j",
						"name": "filter",
						"type": "text",
						"required": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "r9wpo5g3",
						"name": "sort",
						"type": "text",
						"required": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "tnkpwqry",
						"name": "output",
						"type": "file",
						"required": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"maxSize": 2147483648,
							"mimeTypes": [
								"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
								"text/csv"
							],
							"thumbs": [],
							"protected": false
						}
					},
					{
						"system": false,
						"id": "xcaifbrc",
						"name": "format",
						"type": "select",
						"required": true,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"csv",
								"xlsx"
							]
						}
					},
					{
						"system": false,
						"id": "hg2lewuw",
						"name": "ownerId",
						"type": "text",
						"required": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "x0ajpx5w",
						"name": "ownerCollectionName",
						"type": "text",
						"required": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					}
				],
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_BkSyORO` + "`" + ` ON ` + "`" + `pocketexport_exports` + "`" + ` (\n  ` + "`" + `ownerId` + "`" + `,\n  ` + "`" + `ownerCollectionName` + "`" + `\n)"
				],
				"listRule": "ownerId = @request.auth.id && ownerCollectionName = @request.auth.collectionName",
				"viewRule": "ownerId = @request.auth.id && ownerCollectionName = @request.auth.collectionName",
				"createRule": "ownerId = @request.auth.id && ownerCollectionName = @request.auth.collectionName",
				"updateRule": null,
				"deleteRule": "ownerId = @request.auth.id && ownerCollectionName = @request.auth.collectionName",
				"options": {}
			}
		]`

		collections := []*models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			return err
		}

		return daos.New(db).ImportCollections(collections, true, nil)
	}, func(db dbx.Builder) error {
		return nil
	})
}
