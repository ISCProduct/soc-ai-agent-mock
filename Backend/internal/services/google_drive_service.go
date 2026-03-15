package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"

	"golang.org/x/oauth2/google"
)

// GoogleDriveService uploads files to Google Drive using a service account.
type GoogleDriveService struct {
	folderID string
	client   *http.Client
}

type driveFileResponse struct {
	ID      string `json:"id"`
	WebViewLink string `json:"webViewLink"`
}

// NewGoogleDriveService initialises the service from environment variables.
// GOOGLE_SERVICE_ACCOUNT_JSON: service-account credentials JSON (raw string or file path)
// GOOGLE_DRIVE_FOLDER_ID: the Drive folder to upload into
func NewGoogleDriveService() (*GoogleDriveService, error) {
	folderID := os.Getenv("GOOGLE_DRIVE_FOLDER_ID")
	if folderID == "" {
		return nil, errors.New("GOOGLE_DRIVE_FOLDER_ID is not set")
	}

	saJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	if saJSON == "" {
		return nil, errors.New("GOOGLE_SERVICE_ACCOUNT_JSON is not set")
	}

	credBytes := []byte(saJSON)
	// Support file path as fallback
	if _, err := os.Stat(saJSON); err == nil {
		credBytes, err = os.ReadFile(saJSON)
		if err != nil {
			return nil, fmt.Errorf("reading service account file: %w", err)
		}
	}

	cfg, err := google.JWTConfigFromJSON(credBytes, "https://www.googleapis.com/auth/drive.file")
	if err != nil {
		return nil, fmt.Errorf("parsing service account JSON: %w", err)
	}

	return &GoogleDriveService{
		folderID: folderID,
		client:   cfg.Client(context.Background()),
	}, nil
}

// UploadFile uploads data to Google Drive and returns (fileID, webViewLink, error).
func (s *GoogleDriveService) UploadFile(ctx context.Context, fileName, mimeType string, data []byte) (string, string, error) {
	// Build multipart body: metadata part + media part
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	// Part 1: JSON metadata
	metaHeader := textproto.MIMEHeader{}
	metaHeader.Set("Content-Type", "application/json; charset=UTF-8")
	metaPart, err := mw.CreatePart(metaHeader)
	if err != nil {
		return "", "", err
	}
	meta := map[string]interface{}{
		"name":    fileName,
		"parents": []string{s.folderID},
	}
	if err := json.NewEncoder(metaPart).Encode(meta); err != nil {
		return "", "", err
	}

	// Part 2: media
	mediaHeader := textproto.MIMEHeader{}
	mediaHeader.Set("Content-Type", mimeType)
	mediaPart, err := mw.CreatePart(mediaHeader)
	if err != nil {
		return "", "", err
	}
	if _, err := mediaPart.Write(data); err != nil {
		return "", "", err
	}
	mw.Close()

	url := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id,webViewLink"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "multipart/related; boundary="+mw.Boundary())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("drive upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("drive upload failed (%d): %s", resp.StatusCode, string(body))
	}

	var result driveFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decoding drive response: %w", err)
	}

	return result.ID, result.WebViewLink, nil
}
