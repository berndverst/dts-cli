// Package components provides reusable TUI widgets for dts-cli.
package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ResourceTable is a sortable, selectable data table with multi-select support.
type ResourceTable struct {
	*tview.Table
	headers       []string
	selectedRows  map[int]bool
	sortColumn    int
	sortAscending bool
	onSort        func(column int, ascending bool)
	onSelect      func(row int)
}

// NewResourceTable creates a new resource table with column headers.
func NewResourceTable(headers []string) *ResourceTable {
	t := &ResourceTable{
		Table:        tview.NewTable(),
		headers:      headers,
		selectedRows: make(map[int]bool),
		sortColumn:   -1,
	}

	t.SetBorders(false)
	t.SetSelectable(true, false)
	t.SetFixed(1, 0) // Fix header row
	t.SetSeparator(' ')

	// Set header row
	for col, header := range headers {
		cell := tview.NewTableCell(" " + header + " ").
			SetSelectable(false).
			SetTextColor(tcell.ColorAqua).
			SetAttributes(tcell.AttrBold).
			SetAlign(tview.AlignLeft)
		t.SetCell(0, col, cell)
	}

	return t
}

// SetSortHandler sets the callback for sort column changes.
func (t *ResourceTable) SetSortHandler(handler func(column int, ascending bool)) {
	t.onSort = handler
}

// SetSelectHandler sets the callback for row selection (Enter key).
func (t *ResourceTable) SetSelectHandler(handler func(row int)) {
	t.onSelect = handler
	t.SetSelectedFunc(func(row, column int) {
		if row > 0 && handler != nil {
			handler(row - 1) // Adjust for header
		}
	})
}

// ToggleRowSelection toggles multi-select on the given row (Space key).
func (t *ResourceTable) ToggleRowSelection(row int) {
	if row <= 0 {
		return
	}
	dataRow := row - 1
	if t.selectedRows[dataRow] {
		delete(t.selectedRows, dataRow)
		t.updateRowStyle(row, false)
	} else {
		t.selectedRows[dataRow] = true
		t.updateRowStyle(row, true)
	}
}

// SelectAllRows selects or deselects all data rows.
func (t *ResourceTable) SelectAllRows(selectAll bool) {
	for row := 1; row < t.GetRowCount(); row++ {
		dataRow := row - 1
		if selectAll {
			t.selectedRows[dataRow] = true
		} else {
			delete(t.selectedRows, dataRow)
		}
		t.updateRowStyle(row, selectAll)
	}
}

// GetSelectedRows returns the indices of all selected data rows.
func (t *ResourceTable) GetSelectedRows() []int {
	result := make([]int, 0, len(t.selectedRows))
	for row := range t.selectedRows {
		result = append(result, row)
	}
	return result
}

// ClearSelection removes all selections.
func (t *ResourceTable) ClearSelection() {
	for row := range t.selectedRows {
		t.updateRowStyle(row+1, false)
	}
	t.selectedRows = make(map[int]bool)
}

// SetDataRow sets a row of data cells.
func (t *ResourceTable) SetDataRow(row int, cells ...string) {
	tableRow := row + 1 // Account for header
	for col, text := range cells {
		cell := tview.NewTableCell(" " + text + " ").
			SetAlign(tview.AlignLeft)
		t.SetCell(tableRow, col, cell)
	}
}

// SetColoredDataRow sets a row with specific text color.
func (t *ResourceTable) SetColoredDataRow(row int, color tcell.Color, cells ...string) {
	tableRow := row + 1
	for col, text := range cells {
		cell := tview.NewTableCell(" " + text + " ").
			SetAlign(tview.AlignLeft).
			SetTextColor(color)
		t.SetCell(tableRow, col, cell)
	}
}

// ClearData removes all data rows, keeping headers.
func (t *ResourceTable) ClearData() {
	for r := t.GetRowCount() - 1; r > 0; r-- {
		t.RemoveRow(r)
	}
	t.selectedRows = make(map[int]bool)
}

// CycleSort cycles through sort on the given column.
func (t *ResourceTable) CycleSort(column int) {
	if column == t.sortColumn {
		t.sortAscending = !t.sortAscending
	} else {
		t.sortColumn = column
		t.sortAscending = true
	}

	t.updateHeaderIndicators()

	if t.onSort != nil {
		t.onSort(t.sortColumn, t.sortAscending)
	}
}

// SetSortDirection updates the sort direction and header indicator without triggering a sort callback.
func (t *ResourceTable) SetSortDirection(ascending bool) {
	t.sortAscending = ascending
	t.updateHeaderIndicators()
}

func (t *ResourceTable) updateHeaderIndicators() {
	for col, header := range t.headers {
		indicator := " "
		if col == t.sortColumn {
			if t.sortAscending {
				indicator = " ▲"
			} else {
				indicator = " ▼"
			}
		}
		cell := tview.NewTableCell(" " + header + indicator + " ").
			SetSelectable(false).
			SetTextColor(tcell.ColorAqua).
			SetAttributes(tcell.AttrBold).
			SetAlign(tview.AlignLeft)
		t.SetCell(0, col, cell)
	}
}

func (t *ResourceTable) updateRowStyle(tableRow int, selected bool) {
	for col := 0; col < len(t.headers); col++ {
		cell := t.GetCell(tableRow, col)
		if cell != nil {
			if selected {
				cell.SetBackgroundColor(tcell.ColorDarkCyan)
			} else {
				cell.SetBackgroundColor(tcell.ColorDefault)
			}
		}
	}
}

// DataRowCount returns the number of data rows (excluding header).
func (t *ResourceTable) DataRowCount() int {
	count := t.GetRowCount() - 1
	if count < 0 {
		return 0
	}
	return count
}

// NextSortableColumn cycles to the next sortable column.
// skipColumns is a set of column indices to skip (unsortable columns).
func (t *ResourceTable) NextSortableColumn(skipColumns map[int]bool) {
	start := t.sortColumn + 1
	for i := 0; i < len(t.headers); i++ {
		col := (start + i) % len(t.headers)
		if !skipColumns[col] {
			t.CycleSort(col)
			return
		}
	}
}
