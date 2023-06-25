package pocketexport

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/routine"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	// FormatCSV is the csv format
	FormatCSV = "csv"
	// FormatXLSX is the xlsx format
	FormatXLSX = "xlsx"
)

const (
	// ExportCollectionField is the field name for the export collection
	PocketExportCollectionName = "pocketexport_exports"
	// ExportCollectionNameField is the field name for the export collection
	ExportCollectionNameField = "exportCollectionName"
	// OwnerIdField is the field name for the owner id
	OwnerIdField = "ownerId"
	// OwnerCollectionNameField is the field name for the owner collection name
	OwnerCollectionNameField = "ownerCollectionName"
	// FilterField is the field name for the export filter
	FilterField = "filter"
	// SortField is the field name for the export sort
	SortField = "sort"
	// HeadersField is the field name for the export headers
	HeadersField = "headers"
	// FormatField is the field name for the export format
	FormatField = "format"
	// OutputField is the field name for the export output
	OutputField = "output"
)

type RegisterOption func(*registerConfig)

type registerConfig struct {
	generateOutputInBackground bool
	autoDelete                 bool
	autoDeleteDuration         time.Duration
}

var defaultRegisterConfig = registerConfig{
	generateOutputInBackground: false,
	autoDelete:                 true,
	autoDeleteDuration:         time.Hour,
}

// GenerateInBackground sets the generateOutputInBackground option
// if g is true, the export output will be generated in background
func GenerateInBackground(g bool) RegisterOption {
	return func(rc *registerConfig) {
		rc.generateOutputInBackground = g
	}
}

// AutoDelete sets the autoDelete option
// if d is true, the export will be deleted automatically after
// create new export
func AutoDelete(d bool) RegisterOption {
	return func(rc *registerConfig) {
		rc.autoDelete = d
	}
}

// AutoDeleteDuration sets the autoDeleteDuration option
func AutoDeleteDuration(d time.Duration) RegisterOption {
	return func(rc *registerConfig) {
		rc.autoDeleteDuration = d
	}
}

// Register registers the pocketexport app with the core.App
func Register(app core.App, opts ...RegisterOption) error {
	return New(app).Register(opts...)
}

type IPocketExport interface {
	// ValidateAndFill validates the record and fills the export
	// this function should be called after using form validation because
	// it does not fully validate the record
	ValidateAndFill(*models.Record) (*Export, error)
	// GenerateExportOutput generates the output for the export
	GenerateExportOutput(io.Writer, *Export) error
}

type PocketExport struct {
	app core.App
}

// New creates a new pocketexport
func New(app core.App) *PocketExport {
	return &PocketExport{app: app}
}

// ValidateRecord implement PocketExport interface
func (p *PocketExport) ValidateAndFill(r *models.Record) (*Export, error) {
	return p.validateAndFill(r)
}

// GenerateExportOutput implement PocketExport interface
func (p *PocketExport) GenerateExportOutput(dst io.Writer, r *Export) error {
	return p.generateExportOutput(dst, r)
}

// Register implement PocketExport interface
func (p *PocketExport) Register(opts ...RegisterOption) error {
	rc := defaultRegisterConfig
	for _, opt := range opts {
		opt(&rc)
	}

	getFile := func(export *Export) (*filesystem.File, error) {
		buf := bytes.NewBuffer(nil)
		if err := p.GenerateExportOutput(buf, export); err != nil {
			return nil, err
		}

		file, err := filesystem.NewFileFromBytes(buf.Bytes(), export.GetString(OutputField))
		if err != nil {
			return nil, err
		}

		// ensure file name is original name
		file.Name = file.OriginalName
		return file, nil
	}

	// validate export records
	p.app.OnRecordBeforeCreateRequest().Add(func(e *core.RecordCreateEvent) (err error) {
		if e.Record.TableName() != PocketExportCollectionName {
			return nil
		}

		filename := security.RandomString(20)
		switch e.Record.GetString(FormatField) {
		case FormatCSV:
			filename += ".csv"
		case FormatXLSX:
			filename += ".xlsx"
		}

		e.Record.Set(OutputField, filename)
		export, err := p.ValidateAndFill(e.Record)
		if err != nil {
			return err
		}

		if rc.generateOutputInBackground {
			return nil
		}

		file, err := getFile(export)
		if err != nil {
			return err
		}

		// upload file to filesystem
		e.UploadedFiles[OutputField] = []*filesystem.File{file}
		return
	})

	// after create export generate output
	if rc.generateOutputInBackground {
		p.app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
			if e.Record.TableName() != PocketExportCollectionName {
				return nil
			}

			recordId := e.Record.GetId()
			routine.FireAndForget(func() {
				record, err := p.app.Dao().FindRecordById(PocketExportCollectionName, recordId)
				if err != nil {
					log.Printf("pocketexport: find record failed: %v", err)
					return
				}

				export := NewExport(record)
				if err = export.Fill(p.app.Dao()); err != nil {
					log.Printf("pocketexport: fill export failed: %v", err)
					return
				}

				file, err := getFile(export)
				if err != nil {
					log.Printf("pocketexport: upload file failed: %v", err)
					return
				}

				fs, err := p.app.NewFilesystem()
				if err != nil {
					log.Printf("pocketexport: get filesystem failed: %v", err)
					return
				}

				fileKey := record.BaseFilesPath() + "/" + file.Name
				if err = fs.UploadFile(file, fileKey); err != nil {
					log.Printf("pocketexport: upload file failed: %v", err)
				}
			})

			return nil
		})
	}

	// after create delete old exports
	if rc.autoDelete {
		p.app.OnModelAfterCreate().Add(func(e *core.ModelEvent) error {
			if e.Model.TableName() != PocketExportCollectionName {
				return nil
			}

			dao := p.app.Dao()
			records, err := dao.FindRecordsByExpr(
				PocketExportCollectionName,
				dbx.NewExp(
					PocketExportCollectionName+".created <= {:date}",
					dbx.Params{"date": time.Now().UTC().Add(-rc.autoDeleteDuration).Format(types.DefaultDateLayout)},
				),
			)
			if err != nil {
				return err
			}

			for _, r := range records {
				if err := dao.DeleteRecord(r); err != nil {
					return err
				}
			}

			return nil
		})
	}

	return nil
}

type HeaderItem struct {
	FieldName string         `json:"fieldName"`
	Header    string         `json:"header"`
	Timezone  string         `json:"timezone"`
	ValueMap  map[string]any `json:"valueMap"`
}

// Format formats the value.
func (i *HeaderItem) Format(value any) any {
	location, err := time.LoadLocation(i.Timezone)
	if err != nil {
		location = time.UTC
	}

	if v, ok := value.(time.Time); ok {
		value = v.In(location).Format(time.RFC3339)
	}

	if v, ok := value.(types.DateTime); ok {
		value = v.Time().In(location).Format(time.RFC3339)
	}

	if len(i.ValueMap) > 0 {
		if v, ok := i.ValueMap[fmt.Sprintf("%v", value)]; ok {
			value = v
		}
	}

	return value
}

// Export represents an export
type Export struct {
	*models.Record

	exportCollection *models.Collection
	authRecord       *models.Record
	admin            *models.Admin
	headers          []HeaderItem
}

// NewExport creates a new export
func NewExport(r *models.Record) *Export {
	return &Export{Record: r}
}

// Fill fills the export with the export collection, auth record and admin
func (e *Export) Fill(dao *daos.Dao) (err error) {
	var admin *models.Admin
	var authRecord *models.Record
	exportCollection, err := dao.FindCollectionByNameOrId(e.Record.GetString(ExportCollectionNameField))
	if err != nil {
		return err
	}

	if ownerId := e.Record.GetString(OwnerIdField); ownerId != "" {
		if ownerCollectionName := e.Record.GetString(OwnerCollectionNameField); ownerCollectionName != "" {
			authRecord, err = dao.FindRecordById(ownerCollectionName, ownerId)
		} else if OwnerIdField != "" {
			admin, err = dao.FindAdminById(ownerId)
		}

		if err != nil {
			return err
		}
	}

	if err := e.UnmarshalJSONField(HeadersField, &e.headers); err != nil {
		return err
	}

	e.exportCollection = exportCollection
	e.authRecord = authRecord
	e.admin = admin

	return nil
}

// ExportCollection  return the export collection
func (e *Export) ExportCollection() *models.Collection {
	return e.exportCollection
}

// AuthRecord return the auth record, can be null
func (e *Export) AuthRecord() *models.Record {
	return e.authRecord
}

// Admin return the admin, can be null
func (e *Export) Admin() *models.Admin {
	return e.admin
}

// Headers return the headers
func (e *Export) Headers() []HeaderItem {
	return e.headers
}
