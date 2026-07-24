package repository

type URLRecord struct {
	ID          string
	OriginalURL string
	UserID      string
}

type URLSaveResult struct {
	ID        string
	Duplicate bool
}
