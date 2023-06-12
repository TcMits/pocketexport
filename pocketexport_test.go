package pocketexport

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tests"
)

func getExportRecord(t *testing.T, app core.App) *models.Record {
	coll, err := app.Dao().FindCollectionByNameOrId(PocketExportCollectionName)
	if err != nil {
		t.Fatal(err)
	}

	record := models.NewRecord(coll)
	record.Id = "test"
	record.Set(OwnerIdField, "x9fs8mten7zmwcv")
	record.Set(FilterField, `message != ""`)
	record.Set(SortField, "created")
	record.Set(ExportCollectionNameField, "messages")
	record.Set(FormatField, FormatCSV)
	record.Set(HeadersField, []any{
		map[string]any{
			"fieldName": "message",
			"header":    "nội dung",
		},
		map[string]any{
			"fieldName": "created",
			"header":    "ngày tạo",
			"timezone":  "Asia/Ho_Chi_Minh",
		},
		map[string]any{
			"fieldName": "author.email",
			"header":    "thư diện tử",
		},
		map[string]any{
			"fieldName": "author.name",
			"header":    "tên tác giả",
		},
	})

	return record
}

func Test_pocketExport_ValidateAndFill(t *testing.T) {
	testApp, err := tests.NewTestApp("./test_data")
	if err != nil {
		t.Fatal(err)
	}

	exportService := New(testApp)
	record := models.NewRecord(&models.Collection{Name: "test"})
	if _, err := exportService.ValidateAndFill(record); err.(validation.Error).Code() != "validation_is_not_export" {
		t.Fatal(err)
	}

	record = getExportRecord(t, testApp)
	if export, err := exportService.ValidateAndFill(record); err != nil {
		t.Fatal(err)
	} else if export.Admin() == nil {
		t.Fatal("should have admin")
	} else if export.ExportCollection() == nil {
		t.Fatal("should have export")
	} else if export.AuthRecord() != nil {
		t.Fatal("should not have auth record")
	} else if reflect.DeepEqual(
		export.Headers(),
		[]HeaderItem{
			{
				FieldName: "message",
				Header:    "nội dung",
			},
			{
				FieldName: "created",
				Header:    "ngày tạo",
				Timezone:  "Asia/Ho_Chi_Minh",
			},
			{
				FieldName: "author.email",
				Header:    "thư diện tử",
			},
			{
				FieldName: "author.name",
				Header:    "tên tác giả",
			},
		},
	) == false {
		t.Fatal("wrong headers")
	}

	record = getExportRecord(t, testApp)
	record.Set(OwnerIdField, "djh54wc2hpkhfkw")
	record.Set(OwnerCollectionNameField, "users")
	if export, err := exportService.ValidateAndFill(record); err != nil {
		t.Fatal(err)
	} else if export.Admin() != nil {
		t.Fatal("should not have admin")
	} else if export.ExportCollection() == nil {
		t.Fatal("should have export")
	} else if export.AuthRecord() == nil {
		t.Fatal("should have auth record")
	} else if reflect.DeepEqual(
		export.Headers(),
		[]HeaderItem{
			{
				FieldName: "message",
				Header:    "nội dung",
			},
			{
				FieldName: "created",
				Header:    "ngày tạo",
				Timezone:  "Asia/Ho_Chi_Minh",
			},
			{
				FieldName: "author.email",
				Header:    "thư diện tử",
			},
			{
				FieldName: "author.name",
				Header:    "tên tác giả",
			},
		},
	) == false {
		t.Fatal("wrong headers")
	}

	record = getExportRecord(t, testApp)
	record.Set(ExportCollectionNameField, "wrong")
	if _, err := exportService.ValidateAndFill(record); err == nil {
		t.Fatal("should have error")
	}

	record = getExportRecord(t, testApp)
	record.Set(OwnerIdField, "wrong")
	if _, err := exportService.ValidateAndFill(record); err == nil {
		t.Fatal("should have error")
	}

	record = getExportRecord(t, testApp)
	record.Set(HeadersField, []string{"wrong", "headers"})
	if _, err := exportService.ValidateAndFill(record); err == nil {
		t.Fatal("should have error")
	}

	record = getExportRecord(t, testApp)
	record.Set(FilterField, "wrong_field != ''")
	if _, err := exportService.ValidateAndFill(record); err == nil {
		t.Fatal("should have error")
	}

	record = getExportRecord(t, testApp)
	record.Set(HeadersField, []any{
		map[string]any{
			"fieldName": "wrong_field",
			"header":    "nội dung",
		},
	})
	if _, err := exportService.ValidateAndFill(record); err == nil {
		t.Fatal("should have error")
	}

	record = getExportRecord(t, testApp)
	record.Set(HeadersField, []any{
		map[string]any{
			"fieldName": "created",
			"header":    "ngày tạo",
			"timezone":  "wrong",
		},
	})
	if _, err := exportService.ValidateAndFill(record); err == nil {
		t.Fatal("should have error")
	}
}

func Test_pocketExport_GenerateExportOutput(t *testing.T) {
	testApp, err := tests.NewTestApp("./test_data")
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(nil)
	exportService := New(testApp)

	buf = bytes.NewBuffer(nil)
	record := getExportRecord(t, testApp)
	export := NewExport(record)
	if err := export.Fill(testApp.Dao()); err != nil {
		t.Fatal(err)
	}
	if err := exportService.GenerateExportOutput(buf, export); err != nil {
		t.Fatal(err)
	}

	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatal(err)
	}

	expect := "nội dung,ngày tạo,thư diện tử,tên tác giả\n"

	message1, err := testApp.Dao().FindRecordById("messages", "m0emwpt0lnxhm1b")
	if err != nil {
		t.Fatal(err)
	}
	expect += message1.GetString("message") + "," + message1.GetDateTime("created").Time().In(location).Format(time.RFC3339) + ","

	user1, err := testApp.Dao().FindRecordById("users", "vzz4enej24xtni9")
	if err != nil {
		t.Fatal(err)
	}
	expect += user1.GetString("email") + "," + user1.GetString("name") + "\n"

	message2, err := testApp.Dao().FindRecordById("messages", "jywxqa5dv3ps3u7")
	if err != nil {
		t.Fatal(err)
	}
	expect += message2.GetString("message") + "," + message2.GetDateTime("created").Time().In(location).Format(time.RFC3339) + ","

	user2, err := testApp.Dao().FindRecordById("users", "djh54wc2hpkhfkw")
	if err != nil {
		t.Fatal(err)
	}
	expect += user2.GetString("email") + "," + user2.GetString("name") + "\n"

	if buf.String() != expect {
		t.Fatalf("expect:\n%v\ngot:\n%v", expect, buf.String())
	}

	buf = bytes.NewBuffer(nil)
	record.Set(FormatField, FormatXLSX)
	if err := export.Fill(testApp.Dao()); err != nil {
		t.Fatal(err)
	}
	if err := exportService.GenerateExportOutput(buf, export); err != nil {
		t.Fatal(err)
	}
}
