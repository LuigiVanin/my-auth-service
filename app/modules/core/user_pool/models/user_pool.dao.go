package models

// UserPoolUpdateDao is the allow list of updatable columns of entity.UsersPool.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
type UserPoolUpdateDao struct {
	Name      *string
	PublicKey *string
}
