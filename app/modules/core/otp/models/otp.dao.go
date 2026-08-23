package models

// OtpUpdateDao is the allow list of updatable columns of entity.Otp.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
//
// NOTE: this is what replaced the old Save(otp) calls. Save rewrote every
// column of the row, so two concurrent verifications of the same OTP could
// clobber each other; now only the column being changed is written.
type OtpUpdateDao struct {
	Invalidated       *bool
	VerificationCount *int
}
