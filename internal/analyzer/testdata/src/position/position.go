package position

func query() string {
	return `select id from users` // want "SQL literal is not formatted"
}
