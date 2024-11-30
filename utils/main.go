package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/crypto/bcrypt"
)

func GenerateOTP() string {
	otp := rand.Intn(900000) + 100000
	return strconv.Itoa(otp)
}

func GenerateID(length ...int) string {
	idLength := 10
	if len(length) > 0 {
		idLength = length[0]
	}

	id, _ := gonanoid.Generate("qwertyuiopasdfghjklzxcvbnm1234567890_-", idLength)
	return id
}

// VerifyPhoneNumber checks if the phone number is in the format +917569236628
func VerifyPhoneNumber(phone string) bool {
	// Define the regular expression pattern
	phonePattern := `^\+91\d{10}$`
	// Compile the regular expression
	re := regexp.MustCompile(phonePattern)
	// Check if the phone number matches the pattern
	return re.MatchString(phone)
}

// Ensure that single-digit day and month are zero-padded
func FormatDate(input string) string {
	// Split the date into day, month, and year
	parts := strings.Split(input, "/")
	if len(parts) != 3 {
		return input // If the input format is wrong, return as is.
	}

	// Pad single-digit day and month with leading zero if necessary
	if len(parts[0]) == 1 {
		parts[0] = "0" + parts[0]
	}
	if len(parts[1]) == 1 {
		parts[1] = "0" + parts[1]
	}

	// Reconstruct the date string in the desired format "DD/MM/YYYY"
	return strings.Join(parts, "/")
}

// IsValidEmail validates an email address using a regex pattern
func IsValidEmail(email string) bool {
	// Define the regex pattern for a valid email
	// This pattern checks for a general format of "local-part@domain"
	regexPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

	// Compile the regex pattern
	regex := regexp.MustCompile(regexPattern)

	// Check if the email matches the regex pattern
	return regex.MatchString(email)
}

func IsValidURL(url string) bool {
	regex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return regex.MatchString(url)
}

func IsValidIPAddress(ip string) bool {
	regex := regexp.MustCompile(`^(([0-9]{1,3}\.){3}[0-9]{1,3}|([a-fA-F0-9:]+:+)+[a-fA-F0-9]+)$`)
	return regex.MatchString(ip)
}

func IsStrongPassword(password string) bool {
	regex := regexp.MustCompile(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$`)
	return regex.MatchString(password)
}

func Slugify(text string) string {
	regex := regexp.MustCompile(`[^\w\s-]`)
	text = regex.ReplaceAllString(text, "")
	text = strings.ToLower(strings.TrimSpace(text))
	return strings.ReplaceAll(text, " ", "-")
}

func CamelToSnake(s string) string {
	regex := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	return strings.ToLower(regex.ReplaceAllString(s, "${1}_${2}"))
}

func IsValidCreditCard(card string) bool {
	regex := regexp.MustCompile(`^4[0-9]{12}(?:[0-9]{3})?$`) // Example for Visa cards
	return regex.MatchString(card)
}

func SanitizeInput(input string) string {
	regex := regexp.MustCompile(`[<>\"'%;()&+]`)
	return regex.ReplaceAllString(input, "")
}

func FileChecksum(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}

func HumanReadableSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	floatSize := float64(size)
	for floatSize >= 1024 && i < len(units)-1 {
		floatSize /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", floatSize, units[i])
}

func IsValidHexColor(color string) bool {
	regex := regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	return regex.MatchString(color)
}

// Function to parse date with single-digit handling
func ParseDate(input string) (time.Time, error) {
	// Format the input date to ensure it's in "DD/MM/YYYY"
	formattedDate := FormatDate(input)

	// Parse the formatted date
	date, err := time.Parse("02/01/2006", formattedDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing date: %v", err)
	}
	return date, nil
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func CheckPasswordHash(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func Contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func GetNow() string {
	return time.Now().String()
}
