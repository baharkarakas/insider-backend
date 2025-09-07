package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(p string) (string, error) {
    b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
    return string(b), err
}

// alias: VerifyPassword adıyla da kullanılabilsin
func VerifyPassword(plain, hash string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// eski çağrılar için hâlâ dursun
func ComparePassword(plain, hash string) error {
    return VerifyPassword(plain, hash)
}
