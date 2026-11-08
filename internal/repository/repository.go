package repository

// URLRecord хранит ID ссылки, оригинальный URL, владельца и признак удаления.
type URLRecord struct {
	// ID — короткий идентификатор без базового URL.
	ID string
	// OriginalURL — оригинальный URL.
	OriginalURL string
	// UserID — идентификатор владельца ссылки.
	UserID string
	// IsDeleted — признак логического удаления.
	IsDeleted bool
}

// URLSaveResult содержит фактический ID и признак ранее сохранённого URL.
type URLSaveResult struct {
	// ID — короткий идентификатор без базового URL.
	ID string
	// Duplicate — признак повторного сохранения оригинального URL.
	Duplicate bool
}

// URLDeleteRecord задаёт пару ID ссылки и пользователя, имеющего право её удалить.
type URLDeleteRecord struct {
	// ID — короткий идентификатор без базового URL.
	ID string
	// UserID — идентификатор владельца ссылки.
	UserID string
}
