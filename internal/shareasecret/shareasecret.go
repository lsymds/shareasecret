package shareasecret

import "fmt"

// Application is a wrapper/container for the "ShareASecret" project. All jobs and entry points hang off of this
// struct.
type Application struct {
	db *database
}

// NewApplication initializes the Application struct which provides access to all available components of the project.
func NewApplication(connectionString string) (*Application, error) {
	db, err := newSqliteDatabase(connectionString)
	if err != nil {
		return nil, fmt.Errorf("new db: %w", err)
	}

	return &Application{db}, nil
}
