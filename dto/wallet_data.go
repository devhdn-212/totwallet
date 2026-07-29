package dto

type WalletData struct {
	Username  string `json:"username"`
	Token     string `json:"token"`
	Nama      string `json:"nama"`
	Saldo     string `json:"saldo"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_datetime"`
	UpdateAt  string `json:"updated_datetime"`
}

// WalletBalanceData adalah response minimal buat API publik /api/public/balance —
// sengaja tidak ikut expose token/status/timestamp ke website game eksternal.
type WalletBalanceData struct {
	Username string `json:"username"`
	Balance  string `json:"balance"`
}

type CreateWalletRequest struct {
	Username string `json:"username" validate:"required,min=4,max=30,lowercase,alphanum"`
	Password string `json:"password" validate:"required,min=6"`
	Nama     string `json:"nama" validate:"required,max=50"`
}

type UpdateWalletRequest struct {
	Username string `json:"-"`
	Nama     string `json:"nama" validate:"required,max=50"`
	Status   string `json:"status" validate:"required,max=1"`
}

// MemberSaveRequest dipakai satu endpoint POST /api/member/save buat create & edit
// sekaligus (dibedakan lewat field Type: "New" / "Edit"), mengikuti pola AdminSave.
type MemberSaveRequest struct {
	Type     string `json:"type" validate:"required"`
	Username string `json:"username" validate:"required,min=4,max=30,lowercase,alphanum"`
	Password string `json:"password"`
	Nama     string `json:"nama" validate:"required,max=50"`
	Status   string `json:"status" validate:"required,max=1"`
}
