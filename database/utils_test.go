package database

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structs
type TestUser struct {
	TableName string `karma_table:"users" json:"-"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

type TestUserWithPointer struct {
	TableName string  `karma_table:"users" json:"-"`
	Id        int     `json:"id"`
	Name      *string `json:"name"`
	Email     string  `json:"email"`
}

type TestStore struct {
	OauthToken   string `json:"oauth_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

type TestIntegrationStore struct {
	TableName string `karma_table:"integration_store" json:"-"`
	Id        string `json:"id"`
	Uid       string `json:"uid"`
	Iid       string `json:"iid"`
	Store     any    `json:"store" db:"store"`
}

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db := sqlx.NewDb(mockDB, "postgres")
	return db, mock
}

func TestInsertStruct(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	tests := []struct {
		name        string
		tableName   string
		data        any
		expectError bool
		setupMock   func()
	}{
		{
			name:      "successful insert simple struct",
			tableName: "users",
			data: &TestUser{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			expectError: false,
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:      "successful insert with any field",
			tableName: "integration_store",
			data: &TestIntegrationStore{
				Id:  "test-id",
				Uid: "user-123",
				Iid: "linkedin",
				Store: TestStore{
					OauthToken:   "token123",
					RefreshToken: "refresh123",
					ExpiresAt:    1640995200,
					TokenType:    "Bearer",
					Scope:        "read",
				},
			},
			expectError: false,
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO integration_store`).
					WithArgs("test-id", "user-123", "linkedin", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:      "successful insert with pointer field",
			tableName: "users",
			data: &TestUserWithPointer{
				Id:    1,
				Name:  stringPtr("Jane Doe"),
				Email: "jane@example.com",
			},
			expectError: false,
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs(1, "Jane Doe", "jane@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:      "successful insert with nil pointer field",
			tableName: "users",
			data: &TestUserWithPointer{
				Id:    2,
				Name:  nil,
				Email: "user@example.com",
			},
			expectError: false,
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs(2, nil, "user@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "error: nil data",
			tableName:   "users",
			data:        nil,
			expectError: true,
			setupMock:   func() {},
		},
		{
			name:        "error: non-pointer data",
			tableName:   "users",
			data:        TestUser{},
			expectError: true,
			setupMock:   func() {},
		},
		{
			name:        "error: nil pointer",
			tableName:   "users",
			data:        (*TestUser)(nil),
			expectError: true,
			setupMock:   func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := InsertStruct(db, tt.tableName, tt.data)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInsertTrxStruct(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Beginx()
	require.NoError(t, err)

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := &TestUser{
		Name:  "Transaction User",
		Email: "tx@example.com",
	}

	err = InsertTrxStruct(tx, "users", user)
	assert.NoError(t, err)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateStruct(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	tests := []struct {
		name           string
		tableName      string
		data           any
		conditionField string
		conditionValue any
		expectError    bool
		setupMock      func()
	}{
		{
			name:      "successful update",
			tableName: "users",
			data: &TestUser{
				Name:  "Updated Name",
				Email: "updated@example.com",
			},
			conditionField: "name",
			conditionValue: "Updated Name",
			expectError:    false,
			setupMock: func() {
				mock.ExpectExec(`UPDATE users SET`).
					WithArgs(sqlmock.AnyArg(), "Updated Name").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:      "successful update with any field",
			tableName: "integration_store",
			data: &TestIntegrationStore{
				Id:  "test-id",
				Uid: "user-123",
				Iid: "linkedin",
				Store: TestStore{
					OauthToken: "updated_token",
					ExpiresAt:  1640995200,
				},
			},
			conditionField: "id",
			conditionValue: "test-id",
			expectError:    false,
			setupMock: func() {
				mock.ExpectExec(`UPDATE integration_store SET`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "test-id").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:           "error: nil data",
			tableName:      "users",
			data:           nil,
			conditionField: "name",
			conditionValue: "test",
			expectError:    true,
			setupMock:      func() {},
		},
		{
			name:           "error: non-pointer data",
			tableName:      "users",
			data:           TestUser{},
			conditionField: "name",
			conditionValue: "test",
			expectError:    true,
			setupMock:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := UpdateStruct(db, tt.tableName, tt.data, tt.conditionField, tt.conditionValue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateTrxStruct(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Beginx()
	require.NoError(t, err)

	mock.ExpectExec(`UPDATE users SET`).
		WithArgs(sqlmock.AnyArg(), "Updated Transaction User").
		WillReturnResult(sqlmock.NewResult(0, 1))

	user := &TestUser{
		Name:  "Updated Transaction User",
		Email: "updated_tx@example.com",
	}

	err = UpdateTrxStruct(tx, "users", user, "name", "Updated Transaction User")
	assert.NoError(t, err)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParseRows(t *testing.T) {
	tests := []struct {
		name        string
		dest        any
		expectError bool
		setupRows   func() *sql.Rows
		expected    any
	}{
		{
			name:        "error: destination not a pointer",
			dest:        []TestUser{},
			expectError: true,
			setupRows:   func() *sql.Rows { return nil },
		},
		{
			name:        "error: destination not a slice",
			dest:        &TestUser{},
			expectError: true,
			setupRows:   func() *sql.Rows { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupRows != nil {
				rows := tt.setupRows()
				err := ParseRows(rows, tt.dest)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestFetchColumnNames(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	columns := []string{"id", "name", "email", "created_at", "updated_at"}
	rows := sqlmock.NewRows([]string{"column_name"})
	for _, col := range columns {
		rows.AddRow(col)
	}

	mock.ExpectQuery(`SELECT column_name FROM information_schema.columns WHERE table_name = \$1`).
		WithArgs("users").
		WillReturnRows(rows)

	result, err := FetchColumnNames(db, "users")
	assert.NoError(t, err)
	assert.Equal(t, columns, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CamelCase", "camel_case"},
		{"XMLHttpRequest", "x_m_l_http_request"},
		{"Id", "id"},
		{"UserID", "user_i_d"},
		{"simpleWord", "simple_word"},
		{"", ""},
		{"A", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := camelToSnake(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"snake_case", "SnakeCase"},
		{"simple_word", "SimpleWord"},
		{"", ""},
		{"single", "Single"},
		{"multiple_under_scores", "MultipleUnderScores"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := snakeToCamel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringToFloat32(t *testing.T) {
	tests := []struct {
		input       string
		expected    float32
		expectError bool
	}{
		{"123.45", 123.45, false},
		{"0", 0, false},
		{"-123.45", -123.45, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := stringToFloat32(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{1, "$1"},
		{3, "$1, $2, $3"},
		{5, "$1, $2, $3, $4, $5"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.n)), func(t *testing.T) {
			result := placeholders(tt.n)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test struct that follows all the rules
type CompleteUser struct {
	TableName string         `karma_table:"users" json:"-"`
	Id        int            `json:"id"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	Tags      []string       `json:"tags" db:"tags"`
	Metadata  map[string]any `json:"metadata" db:"metadata"`
	Profile   UserProfile    `json:"profile" db:"profile"`
	Settings  *UserSettings  `json:"settings" db:"settings"`
}

type UserProfile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Bio       string `json:"bio"`
}

type UserSettings struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
}

func TestCompleteWorkflow(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	// Test data
	user := &CompleteUser{
		Id:       1,
		Username: "johndoe",
		Email:    "john@example.com",
		Tags:     []string{"developer", "golang"},
		Metadata: map[string]any{
			"last_login": "2023-01-01",
			"preferences": map[string]string{
				"language": "en",
				"timezone": "UTC",
			},
		},
		Profile: UserProfile{
			FirstName: "John",
			LastName:  "Doe",
			Bio:       "Software Developer",
		},
		Settings: &UserSettings{
			Theme:         "dark",
			Notifications: true,
		},
	}

	// Test Insert
	t.Run("insert complete user", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(1, "johndoe", "john@example.com", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := InsertStruct(db, "users", user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Test Update
	t.Run("update complete user", func(t *testing.T) {
		mock.ExpectExec(`UPDATE users SET`).
			WithArgs("johndoe", "john@example.com", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := UpdateStruct(db, "users", user, "id", 1)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Test with nil pointer field
	t.Run("insert user with nil settings", func(t *testing.T) {
		userWithNilSettings := &CompleteUser{
			Id:       2,
			Username: "janedoe",
			Email:    "jane@example.com",
			Tags:     []string{"designer"},
			Metadata: map[string]any{"role": "designer"},
			Profile: UserProfile{
				FirstName: "Jane",
				LastName:  "Doe",
			},
			Settings: nil, // nil pointer
		}

		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(2, "janedoe", "jane@example.com", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := InsertStruct(db, "users", userWithNilSettings)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

// Custom value type for testing sqlmock with any arguments
type AnyValue struct{}

func (a AnyValue) Match(v driver.Value) bool {
	return true
}
