package services

type IHashService interface {
	HashText(text string, salt string) (string, error)
	Compare(text string, hash string) (bool, error)
}
