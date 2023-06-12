package pocketexport

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/resolvers"
	"github.com/pocketbase/pocketbase/tools/search"
	"github.com/xuri/excelize/v2"
)

// GenerateExportOutput generates the export output.
func (s *pocketExport) generateExportOutput(dst io.Writer, export *Export) (err error) {
	filter := export.GetString(FilterField)
	sort := export.GetString(SortField)

	switch export.GetString(FormatField) {
	case FormatCSV:
		err = s.generateExportCSVOutput(dst, filter, sort, export)
	case FormatXLSX:
		err = s.generateExportXLSXOutput(dst, filter, sort, export)
	}

	return
}

// generateExportGetRecordValue is used to get the value of a record.
func (s *pocketExport) generateExportGetRecordValue(r *models.Record, item *HeaderItem, splitKey []string) any {
	nestedRecord := r
	lenSplitKey := len(splitKey)

	// go to the nested record
	for k := range splitKey {
		if k == lenSplitKey-1 || nestedRecord == nil {
			break
		}

		r, ok := nestedRecord.Expand()[splitKey[k]]
		if !ok || r == nil {
			nestedRecord = nil
			continue
		}

		nestedRecord, _ = r.(*models.Record)
	}

	// cannot find the nested record
	if nestedRecord == nil {
		return ""
	}

	// get the value
	return item.Format(nestedRecord.Get(splitKey[lenSplitKey-1]))
}

// generateExportGetHeaderSplitMap return the header split map from header map.
func (s *pocketExport) generateExportGetHeaderSplitMap(headerMap []HeaderItem) map[string][]string {
	headerSplitMap := make(map[string][]string, len(headerMap))

	for i := range headerMap {
		item := &(headerMap)[i]
		splitKey := strings.Split(item.FieldName, ".")
		headerSplitMap[item.FieldName] = splitKey
	}

	return headerSplitMap
}

// generateExportGetExpandsFromHeaderSplitMap return the expands from header split map.
func (s *pocketExport) generateExportGetExpandsFromHeaderSplitMap(headerSplitMap map[string][]string) []string {
	expands := make([]string, 0, len(headerSplitMap))

	for _, splitKey := range headerSplitMap {
		expands = append(expands, strings.Join(splitKey[:len(splitKey)-1], "."))
	}

	return expands
}

// generateExportOutputRecords generates the export output records.
func (s *pocketExport) generateExportOutputRecords(
	records *[]*models.Record,
	filter string,
	sort string,
	export *Export,
	page int,
) error {
	dao := s.app.Dao()

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

	searchProvider := search.NewProvider(fieldResolver).
		Query(dao.RecordQuery(export.ExportCollection())).
		Page(page).
		PerPage(1000)

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

	_, err := searchProvider.Exec(records)
	if err != nil {
		return err
	}

	return nil
}

// generateExportCSVOutput generates the export csv output.
func (s *pocketExport) generateExportCSVOutput(
	buffer io.Writer,
	filter string,
	sort string,
	export *Export,
) error {
	csvWriter := csv.NewWriter(buffer)
	headers := export.Headers()

	// Write headers
	{
		headerStr := make([]string, 0, len(headers))
		for i := range headers {
			item := &(headers)[i]
			headerStr = append(headerStr, item.Header)
		}

		csvWriter.WriteAll([][]string{headerStr})
	}

	// Write records
	{
		records := make([]*models.Record, 0, 1000)
		row := make([]string, len(headers))
		page := 1
		headerSplitMap := s.generateExportGetHeaderSplitMap(headers)
		expands := s.generateExportGetExpandsFromHeaderSplitMap(headerSplitMap)

		for {
			if err := s.generateExportOutputRecords(&records, filter, sort, export, page); err != nil {
				return err
			}

			if err := apis.EnrichRecords(
				&exportEchoContext{export: export},
				s.app.Dao(),
				records,
				expands...,
			); err != nil {
				return err
			}

			for _, record := range records {
				for i := range headers {
					item := &(headers)[i]
					splitKey := headerSplitMap[item.FieldName]
					value := s.generateExportGetRecordValue(record, item, splitKey)
					row[i] = fmt.Sprintf("%v", value)
				}

				csvWriter.WriteAll([][]string{row})
			}

			records = records[:0]
			page += 1

			if len(records) < 1000 {
				break
			}
		}
	}

	return nil
}

// generateExportCSVOutput generates the export xlsx output.
func (s *pocketExport) generateExportXLSXOutput(
	buffer io.Writer,
	filter string,
	sort string,
	export *Export,
) error {
	headers := export.Headers()
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	xlsxWriter, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		return err
	}
	xlsxRowIndex := 1

	// Write headers
	{
		headerStr := make([]any, 0, len(headers))
		for i := range headers {
			item := &(headers)[i]
			headerStr = append(headerStr, item.Header)
		}

		cell, err := excelize.CoordinatesToCellName(1, xlsxRowIndex)
		if err != nil {
			return err
		}

		if err := xlsxWriter.SetRow(cell, headerStr); err != nil {
			return err
		}

		xlsxRowIndex += 1
	}

	// Write records
	{
		records := make([]*models.Record, 0, 1000)
		row := make([]any, len(headers))
		page := 1
		headerSplitMap := s.generateExportGetHeaderSplitMap(headers)
		expands := s.generateExportGetExpandsFromHeaderSplitMap(headerSplitMap)

		for {
			if err := s.generateExportOutputRecords(&records, filter, sort, export, page); err != nil {
				return err
			}

			if err := apis.EnrichRecords(
				&exportEchoContext{export: export},
				s.app.Dao(),
				records,
				expands...,
			); err != nil {
				return err
			}

			for _, record := range records {
				for i := range headers {
					item := &(headers)[i]
					splitKey := headerSplitMap[item.FieldName]
					row[i] = s.generateExportGetRecordValue(record, item, splitKey)
				}

				cell, err := excelize.CoordinatesToCellName(1, xlsxRowIndex)
				if err != nil {
					return err
				}

				if err := xlsxWriter.SetRow(cell, row); err != nil {
					return err
				}

				xlsxRowIndex += 1
			}

			records = records[:0]
			page += 1

			if len(records) < 1000 {
				break
			}
		}
	}

	if err := xlsxWriter.Flush(); err != nil {
		return err
	}

	return f.Write(buffer)
}

// fake echo context for generating export output
type exportEchoContext struct {
	echo.Context

	export *Export
}

// Get returns the value of the given key
// It will return the request data if the key is apis.ContextRequestDataKey
func (c *exportEchoContext) Get(key string) interface{} {
	if key == apis.ContextRequestDataKey {
		return &models.RequestData{
			Method:     http.MethodGet,
			Query:      map[string]any{},
			Data:       map[string]any{},
			Headers:    map[string]any{},
			AuthRecord: c.export.AuthRecord(),
			Admin:      c.export.Admin(),
		}
	}

	return c.Get(key)
}

// QueryParam returns the query param of the given key
// It will return an empty string
func (c *exportEchoContext) QueryParam(key string) string {
	return ""
}
