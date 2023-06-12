package pocketexport

import (
	"net/http"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/resolvers"
	"github.com/pocketbase/pocketbase/tools/search"
)

var (
	errInvalidHeaders = validation.NewError("validation_invalid_headers", "invalid headers")

	ErrIsNotExport = validation.NewError("validation_is_not_export", "is not export")
)

// validateAndFill validates the record and fills the export
func (s *pocketExport) validateAndFill(r *models.Record) (*Export, error) {
	if r.TableName() != PocketExportCollectionName {
		return nil, ErrIsNotExport
	}

	dao := s.app.Dao()
	export := NewExport(r)
	if err := export.Fill(dao); err != nil {
		// see fill function for error details
		return nil, validation.Errors{
			ExportCollectionNameField: err,
			HeadersField:              err,
			OwnerIdField:              err,
			OwnerCollectionNameField:  err,
		}
	}

	filter := r.GetString(FilterField)
	sort := r.GetString(SortField)

	fieldResolver := resolvers.NewRecordFieldResolver(
		dao,
		export.ExportCollection(),
		&models.RequestData{
			Method:     http.MethodGet,
			Query:      map[string]any{},
			Data:       map[string]any{},
			Headers:    map[string]any{},
			AuthRecord: export.AuthRecord(),
			Admin:      export.Admin(),
		},
		false,
	)

	// validate filter and sort
	searchProvider := search.NewProvider(fieldResolver).
		Query(dao.RecordQuery(export.ExportCollection())).
		Page(1).
		PerPage(1)

	if filter != "" {
		searchProvider.AddFilter(search.FilterData(filter))
	}

	if sort != "" {
		for _, sortField := range search.ParseSortFromString(sort) {
			searchProvider.AddSort(sortField)
		}
	}

	// ensure that the user has access to the collection
	if export.Admin() == nil && export.ExportCollection().ListRule != nil {
		searchProvider.AddFilter(search.FilterData(*export.ExportCollection().ListRule))
	}

	if _, err := searchProvider.Exec(&[]*models.Record{}); err != nil {
		return nil, validation.Errors{
			FilterField: err,
			SortField:   err,
		}
	}

	// validate headers
	headers := export.Headers()
	for i := range headers {
		item := &headers[i]
		result, err := fieldResolver.Resolve(item.FieldName)
		if err != nil {
			return nil, validation.Errors{HeadersField: err}
		}

		// we don't want to allow subquery in header
		if result.MultiMatchSubQuery != nil {
			return nil, validation.Errors{HeadersField: errInvalidHeaders}
		}

		// validate timezone
		if item.Timezone == "" {
			continue
		}

		if _, err := time.LoadLocation(item.Timezone); err != nil {
			return nil, validation.Errors{HeadersField: err}
		}
	}

	return export, nil
}
