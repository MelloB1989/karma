package database

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ComplexTestStruct struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Email       *string                `json:"email"`
	Age         int                    `json:"age"`
	Balance     float64                `json:"balance"`
	IsActive    bool                   `json:"isActive"`
	Metadata    map[string]interface{} `json:"metadata" db:"jsonb"`
	Tags        []string               `json:"tags" db:"jsonb"`
	Settings    interface{}            `json:"settings" db:"jsonb"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   *time.Time             `json:"updatedAt"`
	Score       *float32               `json:"score"`
	Description *string                `json:"description"`
}

type NestedStruct struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zipCode"`
}

type User struct {
	UserID    int      `json:"userId"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Address   Address  `json:"address" db:"jsonb"`
	Phones    []string `json:"phones" db:"jsonb"`
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "userId", "user_id"},
		{"multiple words", "firstName", "first_name"},
		{"already snake", "user_id", "user_id"},
		{"single char", "a", "a"},
		{"empty", "", ""},
		{"uppercase start", "UserId", "user_id"},
		{"consecutive caps", "HTTPServer", "h_t_t_p_server"},
		{"all caps", "ID", "i_d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := camelToSnake(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "user_id", "userId"},
		{"multiple words", "first_name", "firstName"},
		{"already camel", "userId", "userId"},
		{"single word", "user", "user"},
		{"empty", "", ""},
		{"single char", "a", "a"},
		{"three words", "user_first_name", "userFirstName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snakeToCamel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "user_id", "UserId"},
		{"multiple words", "first_name", "FirstName"},
		{"single word", "user", "User"},
		{"empty", "", ""},
		{"three words", "user_first_name", "UserFirstName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snakeToPascal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected string
	}{
		{"zero", 0, ""},
		{"one", 1, "$1"},
		{"three", 3, "$1, $2, $3"},
		{"five", 5, "$1, $2, $3, $4, $5"},
		{"ten", 10, "$1, $2, $3, $4, $5, $6, $7, $8, $9, $10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := placeholders(tt.n)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegisterType(t *testing.T) {
	RegisterType("testField", NestedStruct{})

	registeredType, exists := TypeRegistry["testField"]
	require.True(t, exists)
	assert.Equal(t, reflect.TypeOf(NestedStruct{}), registeredType)
}

func TestInferTypeFromJSON(t *testing.T) {
	RegisterType("nestedField", NestedStruct{})

	tests := []struct {
		name      string
		data      []byte
		fieldName string
		wantType  string
		wantErr   bool
	}{
		{
			name:      "registered type",
			data:      []byte(`{"field1":"test","field2":42}`),
			fieldName: "nestedField",
			wantType:  "NestedStruct",
			wantErr:   false,
		},
		{
			name:      "map fallback",
			data:      []byte(`{"key":"value","num":123}`),
			fieldName: "unknownField",
			wantType:  "map",
			wantErr:   false,
		},
		{
			name:      "array fallback",
			data:      []byte(`[1,2,3]`),
			fieldName: "arrayField",
			wantType:  "slice",
			wantErr:   false,
		},
		{
			name:      "invalid json",
			data:      []byte(`{invalid`),
			fieldName: "badField",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := inferTypeFromJSON(tt.data, tt.fieldName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			resultValue := reflect.ValueOf(result)

			switch tt.wantType {
			case "NestedStruct":
				assert.Equal(t, "NestedStruct", reflect.TypeOf(result).Name())
			case "map":
				assert.Equal(t, reflect.Map, resultValue.Kind())
			case "slice":
				assert.Equal(t, reflect.Slice, resultValue.Kind())
			}
		})
	}
}

func TestParseRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tests := []struct {
		name        string
		setupMock   func(sqlmock.Sqlmock)
		dest        interface{}
		expectError bool
		validate    func(*testing.T, interface{})
	}{
		{
			name: "parse simple struct",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "age"}).
					AddRow(1, "John Doe", 30).
					AddRow(2, "Jane Smith", 25)
				m.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			dest: &[]struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			expectError: false,
			validate: func(t *testing.T, dest interface{}) {
				result := dest.(*[]struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Age  int    `json:"age"`
				})
				assert.Len(t, *result, 2)
				assert.Equal(t, 1, (*result)[0].ID)
				assert.Equal(t, "John Doe", (*result)[0].Name)
				assert.Equal(t, 30, (*result)[0].Age)
			},
		},
		{
			name: "parse with snake_case columns",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"user_id", "first_name", "last_name"}).
					AddRow(1, "John", "Doe")
				m.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			dest: &[]struct {
				UserID    int    `json:"userId"`
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			}{},
			expectError: false,
			validate: func(t *testing.T, dest interface{}) {
				result := dest.(*[]struct {
					UserID    int    `json:"userId"`
					FirstName string `json:"firstName"`
					LastName  string `json:"lastName"`
				})
				assert.Len(t, *result, 1)
				assert.Equal(t, 1, (*result)[0].UserID)
				assert.Equal(t, "John", (*result)[0].FirstName)
			},
		},
		{
			name: "parse with nil pointer fields",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email"}).
					AddRow(1, "John", nil)
				m.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			dest: &[]struct {
				ID    int     `json:"id"`
				Name  string  `json:"name"`
				Email *string `json:"email"`
			}{},
			expectError: false,
			validate: func(t *testing.T, dest interface{}) {
				result := dest.(*[]struct {
					ID    int     `json:"id"`
					Name  string  `json:"name"`
					Email *string `json:"email"`
				})
				assert.Len(t, *result, 1)
				assert.Nil(t, (*result)[0].Email)
			},
		},
		{
			name: "parse with JSON fields",
			setupMock: func(m sqlmock.Sqlmock) {
				metadata := `{"key":"value","num":42}`
				tags := `["tag1","tag2","tag3"]`
				rows := sqlmock.NewRows([]string{"id", "name", "metadata", "tags"}).
					AddRow(1, "Test", []byte(metadata), []byte(tags))
				m.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			dest: &[]struct {
				ID       int                    `json:"id"`
				Name     string                 `json:"name"`
				Metadata map[string]interface{} `json:"metadata" db:"jsonb"`
				Tags     []string               `json:"tags" db:"jsonb"`
			}{},
			expectError: false,
			validate: func(t *testing.T, dest interface{}) {
				result := dest.(*[]struct {
					ID       int                    `json:"id"`
					Name     string                 `json:"name"`
					Metadata map[string]interface{} `json:"metadata" db:"jsonb"`
					Tags     []string               `json:"tags" db:"jsonb"`
				})
				assert.Len(t, *result, 1)
				assert.Equal(t, "value", (*result)[0].Metadata["key"])
				assert.Len(t, (*result)[0].Tags, 3)
				assert.Equal(t, "tag1", (*result)[0].Tags[0])
			},
		},
		{
			name: "parse with byte slice to string conversion",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "description"}).
					AddRow(1, []byte("byte array description"))
				m.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			dest: &[]struct {
				ID          int    `json:"id"`
				Description string `json:"description"`
			}{},
			expectError: false,
			validate: func(t *testing.T, dest interface{}) {
				result := dest.(*[]struct {
					ID          int    `json:"id"`
					Description string `json:"description"`
				})
				assert.Len(t, *result, 1)
				assert.Equal(t, "byte array description", (*result)[0].Description)
			},
		},
		{
			name: "invalid destination type",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				m.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			dest:        new(string),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			rows, err := db.Query("SELECT")
			require.NoError(t, err)

			err = ParseRows(rows, tt.dest)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.dest)
				}
			}

			rows.Close()
		})
	}
}

func TestExtractFieldsForInsert(t *testing.T) {
	email := "test@example.com"
	now := time.Now()
	score := float32(95.5)

	tests := []struct {
		name           string
		data           interface{}
		expectedCols   int
		expectError    bool
		validateFields func(*testing.T, []string, []any)
	}{
		{
			name: "simple struct",
			data: &struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: 1, Name: "Test"},
			expectedCols: 2,
			expectError:  false,
			validateFields: func(t *testing.T, cols []string, vals []any) {
				assert.Contains(t, cols, "id")
				assert.Contains(t, cols, "name")
				assert.Equal(t, 1, vals[0])
				assert.Equal(t, "Test", vals[1])
			},
		},
		{
			name: "struct with pointer fields",
			data: &struct {
				ID    int     `json:"id"`
				Email *string `json:"email"`
			}{ID: 1, Email: &email},
			expectedCols: 2,
			expectError:  false,
			validateFields: func(t *testing.T, cols []string, vals []any) {
				assert.Contains(t, cols, "email")
				assert.Equal(t, email, vals[1])
			},
		},
		{
			name: "struct with nil pointer",
			data: &struct {
				ID    int     `json:"id"`
				Email *string `json:"email"`
			}{ID: 1, Email: nil},
			expectedCols: 2,
			expectError:  false,
			validateFields: func(t *testing.T, cols []string, vals []any) {
				assert.Nil(t, vals[1])
			},
		},
		{
			name: "struct with JSON fields",
			data: &struct {
				ID       int                    `json:"id"`
				Metadata map[string]interface{} `json:"metadata" db:"jsonb"`
				Tags     []string               `json:"tags" db:"jsonb"`
			}{
				ID:       1,
				Metadata: map[string]interface{}{"key": "value"},
				Tags:     []string{"tag1", "tag2"},
			},
			expectedCols: 3,
			expectError:  false,
			validateFields: func(t *testing.T, cols []string, vals []any) {
				assert.Contains(t, cols, "metadata")
				assert.Contains(t, cols, "tags")

				var metaMap map[string]interface{}
				err := json.Unmarshal([]byte(vals[1].(string)), &metaMap)
				require.NoError(t, err)
				assert.Equal(t, "value", metaMap["key"])
			},
		},
		{
			name: "complex struct",
			data: &ComplexTestStruct{
				ID:        1,
				Name:      "Complex",
				Email:     &email,
				Age:       30,
				Balance:   1000.50,
				IsActive:  true,
				Metadata:  map[string]interface{}{"role": "admin"},
				Tags:      []string{"vip", "premium"},
				Settings:  map[string]string{"theme": "dark"},
				CreatedAt: now,
				UpdatedAt: &now,
				Score:     &score,
			},
			expectedCols: 13,
			expectError:  false,
		},
		{
			name:         "nil data",
			data:         nil,
			expectedCols: 0,
			expectError:  true,
		},
		{
			name:         "non-pointer",
			data:         struct{ ID int }{ID: 1},
			expectedCols: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, vals, err := extractFieldsForInsert(tt.data)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCols, len(cols))
			assert.Equal(t, tt.expectedCols, len(vals))

			if tt.validateFields != nil {
				tt.validateFields(t, cols, vals)
			}
		})
	}
}

func TestInsertStruct(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	email := "test@example.com"
	now := time.Now()

	tests := []struct {
		name        string
		tableName   string
		data        interface{}
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name:      "insert simple struct",
			tableName: "users",
			data: &struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: 1, Name: "John"},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO users").
					WithArgs(1, "John").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:      "insert with pointer fields",
			tableName: "users",
			data: &struct {
				ID    int     `json:"id"`
				Name  string  `json:"name"`
				Email *string `json:"email"`
			}{ID: 1, Name: "John", Email: &email},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO users").
					WithArgs(1, "John", email).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:      "insert with JSON fields",
			tableName: "users",
			data: &struct {
				ID   int      `json:"id"`
				Name string   `json:"name"`
				Tags []string `json:"tags" db:"jsonb"`
			}{
				ID:   1,
				Name: "John",
				Tags: []string{"admin", "user"},
			},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO users").
					WithArgs(1, "John", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:      "insert user with address",
			tableName: "users",
			data: &User{
				UserID:    1,
				FirstName: "John",
				LastName:  "Doe",
				Address: Address{
					Street:  "123 Main St",
					City:    "Boston",
					ZipCode: "02101",
				},
				Phones: []string{"+1234567890", "+0987654321"},
			},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO users").
					WithArgs(1, "John", "Doe", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:      "insert complex struct",
			tableName: "complex_table",
			data: &ComplexTestStruct{
				ID:        1,
				Name:      "Complex Test",
				Email:     &email,
				Age:       30,
				Balance:   1500.75,
				IsActive:  true,
				Metadata:  map[string]interface{}{"key": "value"},
				Tags:      []string{"tag1", "tag2"},
				CreatedAt: now,
			},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO complex_table").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			err := InsertStruct(sqlxDB, tt.tableName, tt.data)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateStruct(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	email := "updated@example.com"

	tests := []struct {
		name           string
		tableName      string
		data           interface{}
		conditionField string
		conditionValue interface{}
		setupMock      func(sqlmock.Sqlmock)
		expectError    bool
	}{
		{
			name:      "update simple struct",
			tableName: "users",
			data: &struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: 1, Name: "Updated Name"},
			conditionField: "id",
			conditionValue: 1,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE users SET name = \\$1 WHERE id = \\$2").
					WithArgs("Updated Name", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:      "update with multiple fields",
			tableName: "users",
			data: &struct {
				ID    int     `json:"id"`
				Name  string  `json:"name"`
				Email *string `json:"email"`
				Age   int     `json:"age"`
			}{ID: 1, Name: "John Doe", Email: &email, Age: 35},
			conditionField: "id",
			conditionValue: 1,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE users").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:      "update with JSON fields",
			tableName: "users",
			data: &struct {
				ID       int                    `json:"id"`
				Name     string                 `json:"name"`
				Metadata map[string]interface{} `json:"metadata" db:"jsonb"`
			}{
				ID:       1,
				Name:     "John",
				Metadata: map[string]interface{}{"updated": true},
			},
			conditionField: "id",
			conditionValue: 1,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE users").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:      "update user with address",
			tableName: "users",
			data: &User{
				UserID:    1,
				FirstName: "Jane",
				LastName:  "Smith",
				Address: Address{
					Street:  "456 Oak Ave",
					City:    "New York",
					ZipCode: "10001",
				},
				Phones: []string{"+1111111111"},
			},
			conditionField: "userId",
			conditionValue: 1,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE users").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			err := UpdateStruct(sqlxDB, tt.tableName, tt.data, tt.conditionField, tt.conditionValue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func BenchmarkCamelToSnake(b *testing.B) {
	inputs := []string{
		"userId",
		"firstName",
		"userAccountId",
		"HTTPServerAddress",
		"simpleString",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = camelToSnake(inputs[i%len(inputs)])
	}
}

func BenchmarkSnakeToCamel(b *testing.B) {
	inputs := []string{
		"user_id",
		"first_name",
		"user_account_id",
		"http_server_address",
		"simple_string",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snakeToCamel(inputs[i%len(inputs)])
	}
}

func BenchmarkPlaceholders(b *testing.B) {
	sizes := []int{1, 5, 10, 20, 50, 100}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = placeholders(size)
			}
		})
	}
}

func BenchmarkExtractFieldsForInsert(b *testing.B) {
	email := "bench@example.com"
	now := time.Now()
	score := float32(88.5)

	testData := &ComplexTestStruct{
		ID:       1,
		Name:     "Benchmark Test",
		Email:    &email,
		Age:      30,
		Balance:  2500.50,
		IsActive: true,
		Metadata: map[string]interface{}{
			"department": "engineering",
			"level":      5,
			"clearance":  true,
		},
		Tags:      []string{"backend", "golang", "database", "performance"},
		Settings:  map[string]string{"theme": "dark", "language": "en"},
		CreatedAt: now,
		UpdatedAt: &now,
		Score:     &score,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = extractFieldsForInsert(testData)
	}
}

func BenchmarkParseRows(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	metadata := `{"role":"admin","permissions":["read","write","delete"]}`
	tags := `["golang","database","performance","testing"]`

	rows := sqlmock.NewRows([]string{
		"id", "name", "email", "age", "balance", "is_active",
		"metadata", "tags", "settings", "created_at", "updated_at", "score", "description",
	})

	for i := 0; i < 100; i++ {
		email := "user" + string(rune(i)) + "@example.com"
		now := time.Now()
		score := float32(85.5 + float32(i)*0.1)
		desc := "Description for user " + string(rune(i))

		rows.AddRow(
			i+1,
			"User "+string(rune(i)),
			email,
			25+i%40,
			1000.0+float64(i)*10.5,
			i%2 == 0,
			[]byte(metadata),
			[]byte(tags),
			[]byte(`{"theme":"dark"}`),
			now,
			now,
			score,
			desc,
		)
	}

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	dest := &[]ComplexTestStruct{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queryRows, _ := db.Query("SELECT")
		_ = ParseRows(queryRows, dest)
		queryRows.Close()

		*dest = (*dest)[:0]

		if i < b.N-1 {
			mock.ExpectQuery("SELECT").WillReturnRows(rows)
		}
	}
}

func BenchmarkInsertStruct(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	email := "bench@example.com"
	now := time.Now()
	score := float32(92.5)

	testData := &ComplexTestStruct{
		ID:       1,
		Name:     "Benchmark Insert",
		Email:    &email,
		Age:      28,
		Balance:  3000.75,
		IsActive: true,
		Metadata: map[string]interface{}{
			"role":   "developer",
			"team":   "backend",
			"level":  3,
			"active": true,
		},
		Tags:      []string{"golang", "postgresql", "redis", "docker"},
		Settings:  map[string]string{"notifications": "enabled", "theme": "light"},
		CreatedAt: now,
		UpdatedAt: &now,
		Score:     &score,
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = InsertStruct(sqlxDB, "benchmark_table", testData)
	}
}

func BenchmarkUpdateStruct(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	email := "update@example.com"
	now := time.Now()
	score := float32(87.3)

	testData := &ComplexTestStruct{
		ID:       1,
		Name:     "Benchmark Update",
		Email:    &email,
		Age:      32,
		Balance:  4500.25,
		IsActive: false,
		Metadata: map[string]interface{}{
			"role":       "senior",
			"department": "engineering",
			"yearsOfExp": 8,
		},
		Tags:      []string{"leadership", "architecture", "mentoring"},
		Settings:  map[string]string{"autoSave": "true", "darkMode": "true"},
		CreatedAt: now,
		UpdatedAt: &now,
		Score:     &score,
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UpdateStruct(sqlxDB, "benchmark_table", testData, "id", 1)
	}
}

func BenchmarkBuildColumnFieldMap(b *testing.B) {
	elemType := reflect.TypeOf(ComplexTestStruct{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildColumnFieldMap(elemType)
	}
}

func BenchmarkInferTypeFromJSON(b *testing.B) {
	RegisterType("testStruct", NestedStruct{})

	testCases := []struct {
		name      string
		data      []byte
		fieldName string
	}{
		{
			name:      "simple object",
			data:      []byte(`{"key":"value","number":42}`),
			fieldName: "simpleField",
		},
		{
			name:      "nested object",
			data:      []byte(`{"field1":"test","field2":100}`),
			fieldName: "testStruct",
		},
		{
			name:      "array",
			data:      []byte(`["item1","item2","item3","item4"]`),
			fieldName: "arrayField",
		},
		{
			name: "complex object",
			data: []byte(`{
				"department":"engineering",
				"employees":50,
				"active":true,
				"locations":["USA","UK","India"]
			}`),
			fieldName: "complexField",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = inferTypeFromJSON(tc.data, tc.fieldName)
			}
		})
	}
}

func TestFetchColumnNames(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tests := []struct {
		name         string
		tableName    string
		setupMock    func(sqlmock.Sqlmock)
		expectedCols []string
		expectError  bool
	}{
		{
			name:      "fetch columns successfully",
			tableName: "users",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"column_name"}).
					AddRow("id").
					AddRow("name").
					AddRow("email").
					AddRow("created_at")
				m.ExpectQuery("SELECT column_name FROM information_schema.columns").
					WithArgs("users").
					WillReturnRows(rows)
			},
			expectedCols: []string{"id", "name", "email", "created_at"},
			expectError:  false,
		},
		{
			name:      "empty table",
			tableName: "empty_table",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"column_name"})
				m.ExpectQuery("SELECT column_name FROM information_schema.columns").
					WithArgs("empty_table").
					WillReturnRows(rows)
			},
			expectedCols: []string{},
			expectError:  false,
		},
		{
			name:      "database error",
			tableName: "error_table",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT column_name FROM information_schema.columns").
					WithArgs("error_table").
					WillReturnError(sql.ErrConnDone)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			cols, err := FetchColumnNames(sqlxDB, tt.tableName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCols, cols)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTransactionInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	email := "transaction@example.com"
	testData := &struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Email *string `json:"email"`
	}{ID: 1, Name: "Transaction Test", Email: &email}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(1, "Transaction Test", email).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)

	err = InsertTrxStruct(tx, "users", testData)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	email := "updated_transaction@example.com"
	testData := &struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Email *string `json:"email"`
	}{ID: 1, Name: "Updated Transaction", Email: &email}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE users").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)

	err = UpdateTrxStruct(tx, "users", testData, "id", 1)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty struct tags", func(t *testing.T) {
		data := &struct {
			ID       int `json:"id"`
			Internal string
		}{ID: 1, Internal: "should be skipped"}

		cols, vals, err := extractFieldsForInsert(data)
		require.NoError(t, err)
		assert.Equal(t, 1, len(cols))
		assert.Equal(t, 1, len(vals))
	})

	t.Run("json tag with omitempty", func(t *testing.T) {
		data := &struct {
			ID   int    `json:"id"`
			Name string `json:"name,omitempty"`
		}{ID: 1, Name: "test"}

		cols, _, err := extractFieldsForInsert(data)
		require.NoError(t, err)
		assert.Equal(t, 2, len(cols))
		assert.Contains(t, cols, "name")
	})

	t.Run("struct with dash json tag", func(t *testing.T) {
		data := &struct {
			ID       int    `json:"id"`
			Internal string `json:"-"`
		}{ID: 1, Internal: "excluded"}

		cols, _, err := extractFieldsForInsert(data)
		require.NoError(t, err)
		assert.Equal(t, 1, len(cols))
		assert.NotContains(t, cols, "Internal")
	})

	t.Run("deeply nested struct", func(t *testing.T) {
		type Level3 struct {
			Value string `json:"value"`
		}
		type Level2 struct {
			Level3 Level3 `json:"level3"`
		}
		type Level1 struct {
			ID     int    `json:"id"`
			Level2 Level2 `json:"level2" db:"jsonb"`
		}

		data := &Level1{
			ID: 1,
			Level2: Level2{
				Level3: Level3{Value: "deep"},
			},
		}

		cols, vals, err := extractFieldsForInsert(data)
		require.NoError(t, err)
		assert.Equal(t, 2, len(cols))

		var level2Map map[string]interface{}
		err = json.Unmarshal([]byte(vals[1].(string)), &level2Map)
		require.NoError(t, err)
		assert.NotNil(t, level2Map["level3"])
	})

	t.Run("interface with various types", func(t *testing.T) {
		data := &struct {
			ID    int         `json:"id"`
			Value interface{} `json:"value" db:"jsonb"`
		}{ID: 1, Value: map[string]interface{}{"key": "value", "num": 42}}

		cols, vals, err := extractFieldsForInsert(data)
		require.NoError(t, err)
		assert.Equal(t, 2, len(cols))

		var valueMap map[string]interface{}
		err = json.Unmarshal([]byte(vals[1].(string)), &valueMap)
		require.NoError(t, err)
		assert.Equal(t, "value", valueMap["key"])
		assert.Equal(t, float64(42), valueMap["num"])
	})
}

func TestConcurrentOperations(t *testing.T) {
	const goroutines = 10
	const iterations = 100

	t.Run("concurrent camelToSnake", func(t *testing.T) {
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					result := camelToSnake("userId")
					if result != "user_id" {
						t.Errorf("Expected user_id, got %s", result)
					}
				}
				done <- true
			}()
		}

		for i := 0; i < goroutines; i++ {
			<-done
		}
	})

	t.Run("concurrent type registration", func(t *testing.T) {
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				for j := 0; j < iterations; j++ {
					fieldName := "field" + string(rune(id))
					RegisterType(fieldName, NestedStruct{})
				}
				done <- true
			}(i)
		}

		for i := 0; i < goroutines; i++ {
			<-done
		}
	})
}

func TestMemoryAllocation(t *testing.T) {
	email := "memory@example.com"
	now := time.Now()
	score := float32(90.0)

	data := &ComplexTestStruct{
		ID:       1,
		Name:     "Memory Test",
		Email:    &email,
		Age:      30,
		Balance:  5000.0,
		IsActive: true,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
		Tags:      []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
		Settings:  map[string]string{"setting1": "value1", "setting2": "value2"},
		CreatedAt: now,
		UpdatedAt: &now,
		Score:     &score,
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < 1000; i++ {
		_, _, _ = extractFieldsForInsert(data)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocPerOp := float64(m2.TotalAlloc-m1.TotalAlloc) / 1000.0
	t.Logf("Average allocation per operation: %.2f bytes", allocPerOp)
}
