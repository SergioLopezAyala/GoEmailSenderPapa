package emailSender

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("SendEmail", SendEmail)
}

// ContactRequest represents the incoming contact form request structure
type ContactRequest struct {
	Nombre  string `json:"nombre"`
	Empresa string `json:"empresa"`
	Email   string `json:"email"`
	Asunto  string `json:"asunto"`
	Mensaje string `json:"mensaje,omitempty"`
}

// ContactResponse represents the response structure
type ContactResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// SendEmail is the Cloud Function entry point
func SendEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var contactReq ContactRequest
	if err := json.NewDecoder(r.Body).Decode(&contactReq); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateContactRequest(&contactReq); err != nil {
		log.Printf("Validation failed: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := sendContactEmailViaSMTP(&contactReq); err != nil {
		log.Printf("Failed to send email: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to send email: %v", err))
		return
	}

	response := ContactResponse{
		Success: true,
		Message: "Email sent successfully",
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)

	log.Printf("Email sent successfully from: %s (%s)", contactReq.Nombre, contactReq.Email)
}

// validateContactRequest validates all required contact form fields
func validateContactRequest(req *ContactRequest) error {
	req.Nombre = strings.TrimSpace(req.Nombre)
	req.Empresa = strings.TrimSpace(req.Empresa)
	req.Email = strings.TrimSpace(req.Email)
	req.Asunto = strings.TrimSpace(req.Asunto)

	if req.Nombre == "" {
		return fmt.Errorf("'nombre' field is required")
	}
	if req.Empresa == "" {
		return fmt.Errorf("'empresa' field is required")
	}
	if req.Email == "" {
		return fmt.Errorf("'email' field is required")
	}
	if !strings.Contains(req.Email, "@") {
		return fmt.Errorf("'email' field is invalid")
	}
	if req.Asunto == "" {
		return fmt.Errorf("'asunto' field is required")
	}
	return nil
}

func sendContactEmailViaSMTP(req *ContactRequest) error {
	from := os.Getenv("GMAIL_ADDRESS")
	if from == "" {
		return fmt.Errorf("GMAIL_ADDRESS environment variable not set")
	}

	password := os.Getenv("GMAIL_APP_PASSWORD")
	if password == "" {
		return fmt.Errorf("GMAIL_APP_PASSWORD environment variable not set")
	}

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", from, password, smtpHost)

	subject := fmt.Sprintf("Nueva Consulta [%s] de %s - %s", req.Asunto, req.Nombre, req.Empresa)

	// ================================
	// BUILD EMAIL MESSAGE
	// ================================
	msg := fmt.Sprintf(
		"Subject: %s\r\nTo: %s\r\nReply-To: %s\r\n\r\nNombre: %s\r\nEmpresa: %s\r\nEmail: %s\r\nAsunto: %s\r\nMensaje:\r\n%s",
		subject,
		from,
		req.Email,
		req.Nombre,
		req.Empresa,
		req.Email,
		req.Asunto,
		req.Mensaje,
	)

	addr := smtpHost + ":" + smtpPort

	// ================================
	// SEND EMAIL
	// ================================
	err := smtp.SendMail(
		addr,
		auth,
		from,
		[]string{from},
		[]byte(msg),
	)

	if err != nil {
		return fmt.Errorf("smtp error: %w", err)
	}

	return nil
}

// sendErrorResponse sends a JSON error response
func sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := ContactResponse{
		Success: false,
		Message: "Failed to send email",
		Error:   message,
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}
