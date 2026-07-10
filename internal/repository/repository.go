package repository

type URLRecord struct {
	ID          string
	OriginalURL string
}

type URLSaveResult struct {
	ID        string
	Duplicate bool
}
