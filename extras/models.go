package extras

import "time"

type AuthData struct {
	Email   string
	Token   string
	UserKey string
}

type Book struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Author         string    `json:"author"`
	Key            string    `json:"key"`
	AuthorKey      string    `json:"authorKey"`
	ReleaseYear    int       `json:"releaseYear"`
	DateAdded      time.Time `json:"dateAdded"`
	StartReading   time.Time `json:"startReading"`
	FinishReading  time.Time `json:"finishReading"`
	CoverURL       string    `json:"coverURL"`
	MyRating       float32   `json:"myRating"`
	AVGRating      float32   `json:"avgRating"`
	Comment        string    `json:"comment"`
	PageCount      int       `json:"pageCount"`
	CollecionID    string    `json:"collectionID"`
	TempCollection string    `json:"TempCollection,omitempty"`
}

type Collection struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	CreationDate   time.Time `json:"creationDate"`
	ContainedBooks int       `json:"containedBooks"`
	OwnerID        string    `json:"ownerID"`
	Exclusive      bool      `json:"exclusive"`
	ReadCol        bool      `json:"readCol"`
	Editable       bool      `json:"editable"`
}

type MigrationOption int

const (
	GoodReads MigrationOption = iota
)
