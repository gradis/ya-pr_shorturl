package repository

type URLRecord struct {
	ID          string
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

type URLSaveResult struct {
	ID        string
	Duplicate bool
}

type URLDeleteRecord struct {
	ID     string
	UserID string
}
