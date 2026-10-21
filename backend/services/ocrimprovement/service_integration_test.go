package ocrimprovement

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/internal/ocrtest"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/ocr"
	"expense-tracker/backend/services/user"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type object struct {
	data     []byte
	metadata map[string]string
}
type memoryObjects struct {
	objects    map[string]object
	failDelete bool
}

func (m *memoryObjects) PutObject(_ context.Context, p *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	data, _ := io.ReadAll(p.Body)
	m.objects[*p.Key] = object{data, p.Metadata}
	return &s3.PutObjectOutput{}, nil
}
func (m *memoryObjects) HeadObject(_ context.Context, p *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	o, ok := m.objects[*p.Key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &s3.HeadObjectOutput{Metadata: o.metadata}, nil
}
func (m *memoryObjects) CopyObject(_ context.Context, p *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	path, _ := url.PathUnescape(*p.CopySource)
	key := strings.TrimPrefix(path, *p.Bucket+"/")
	o, ok := m.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	m.objects[*p.Key] = o
	return &s3.CopyObjectOutput{}, nil
}
func (m *memoryObjects) DeleteObject(_ context.Context, p *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.failDelete {
		return nil, errors.New("delete unavailable")
	}
	delete(m.objects, *p.Key)
	return &s3.DeleteObjectOutput{}, nil
}
func (m *memoryObjects) ListObjectsV2(_ context.Context, p *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	out := &s3.ListObjectsV2Output{}
	for key := range m.objects {
		if strings.HasPrefix(key, *p.Prefix) {
			if len(out.Contents) >= int(aws.ToInt32(p.MaxKeys)) {
				out.IsTruncated = aws.Bool(true)
				break
			}
			out.Contents = append(out.Contents, s3types.Object{Key: aws.String(key)})
		}
	}
	return out, nil
}

func request(t *testing.T, s *Service, account uuid.UUID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), account)
	require.NoError(t, err)
	router := gin.New()
	s.RegisterRoutes(router.Group(""))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func TestConsentCaptureImmutableSnapshotAndCompleteOptOutPurge(t *testing.T) {
	t.Setenv("OCR_EVIDENCE_CAPTURE_ENABLED", "true")
	t.Setenv("OCR_EVIDENCE_BUCKET", "evidence")
	f := ocrtest.New(t)
	objects := &memoryObjects{objects: map[string]object{}}
	secret := []byte(strings.Repeat("s", 32))
	s := &Service{DB: f.DB, Objects: objects, Bucket: "evidence", Secret: secret, CaptureEnabled: true}
	p, err := s.Preference(t.Context(), f.User.String())
	require.NoError(t, err)
	require.False(t, p.Enabled)
	body := `{"enabled":true,"policyVersion":"ocr-improvement-v1","expectedRevision":0}`
	response := request(t, s, f.User, "PATCH", "/account/ocr-improvement", body)
	require.Equal(t, 403, response.Code)
	_, err = f.DB.Exec(`INSERT INTO user_feature_grant(user_id,feature_key,granted_by_user_id) VALUES($1,'receipt_ocr',$1)`, f.User)
	require.NoError(t, err)
	response = request(t, s, f.User, "PATCH", "/account/ocr-improvement", body)
	require.Equal(t, 200, response.Code, response.Body.String())
	p, err = s.Preference(t.Context(), f.User.String())
	require.NoError(t, err)
	require.True(t, p.Enabled)
	require.EqualValues(t, 2, p.Revision)
	revision, err := user.NewStore(f.DB).GetOCRConsentRevision(t.Context(), f.User.String())
	require.NoError(t, err)
	require.Equal(t, p.Revision, revision)
	response = request(t, s, f.User, "PATCH", "/account/ocr-improvement", body)
	require.Equal(t, 409, response.Code)
	run := uuid.NewString()
	capability := &ocr.CapabilityClaims{ConsentRevision: revision}
	capability.Subject = f.User.String()
	capability.ID = run
	writer := ocr.EvidenceWriter{Client: objects, Bucket: "evidence", Secret: secret}
	draft := ocr.Result{Merchant: ocr.Field{Value: "Shoppers Drug Mart"}, Evidence: &ocr.ExtractionEvidence{Provider: "gemini", Model: ocr.DefaultGeminiModel, AdapterVersion: "gemini-v1", Response: json.RawMessage(`{"candidates":[]}`)}}
	token, err := writer.Store(t.Context(), capability, []byte("processed pixels"), draft, time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, token)
	_, err = ocr.VerifyEvidenceToken(secret, token, f.Other.String(), time.Now())
	require.Error(t, err)
	claims, err := ocr.VerifyEvidenceToken(secret, token, f.User.String(), time.Now())
	require.NoError(t, err)
	require.NoError(t, s.Confirm(t.Context(), f.User.String(), f.Expense, claims))
	key := ocr.EvidencePrefix(f.User.String(), run) + "confirmed.json.gz"
	original := bytes.Clone(objects.objects[key].data)
	zipper, err := gzip.NewReader(bytes.NewReader(original))
	require.NoError(t, err)
	snapshot, err := io.ReadAll(zipper)
	require.NoError(t, err)
	zipper.Close()
	require.Contains(t, string(snapshot), "Shoppers Drug Mart")
	require.NotContains(t, string(snapshot), "password")
	_, err = f.DB.Exec(`UPDATE expense SET provider_name='Corrected later' WHERE id=$1`, f.Expense)
	require.NoError(t, err)
	require.NoError(t, s.Confirm(t.Context(), f.User.String(), f.Expense, claims))
	require.Equal(t, original, objects.objects[key].data)
	response = request(t, s, f.User, "PATCH", "/account/ocr-improvement", `{"enabled":false,"policyVersion":"ocr-improvement-v1","expectedRevision":2}`)
	require.Equal(t, 202, response.Code, response.Body.String())
	require.Error(t, s.Confirm(t.Context(), f.User.String(), f.Expense, claims))
	revision, err = user.NewStore(f.DB).GetOCRConsentRevision(t.Context(), f.User.String())
	require.NoError(t, err)
	require.Zero(t, revision)
	_, err = f.DB.Exec(`UPDATE ocr_improvement_consent SET changed_at=NOW()-INTERVAL '3 minutes' WHERE account_id=$1`, f.User)
	require.NoError(t, err)
	objects.failDelete = true
	require.Error(t, s.PurgeAccount(t.Context(), f.User.String()))
	p, err = s.Preference(t.Context(), f.User.String())
	require.NoError(t, err)
	require.True(t, p.PurgePending)
	objects.failDelete = false
	require.NoError(t, s.PurgeAccount(t.Context(), f.User.String()))
	require.Empty(t, objects.objects)
	p, err = s.Preference(t.Context(), f.User.String())
	require.NoError(t, err)
	require.False(t, p.PurgePending)
	var deleted bool
	require.NoError(t, f.DB.QueryRow(`SELECT is_deleted FROM expense WHERE id=$1`, f.Expense).Scan(&deleted))
	require.False(t, deleted)
	require.NoError(t, s.RequestAccountPurge(t.Context(), f.User.String()))
	p, err = s.Preference(t.Context(), f.User.String())
	require.NoError(t, err)
	require.True(t, p.PurgePending)
	require.False(t, p.Enabled)
}

func TestDarkCaptureOptOutNeedsNoStorageAndDoesNotIssueConsentedCapabilities(t *testing.T) {
	t.Setenv("OCR_EVIDENCE_CAPTURE_ENABLED", "false")
	t.Setenv("OCR_EVIDENCE_BUCKET", "")
	f := ocrtest.New(t)
	s := &Service{DB: f.DB}
	_, err := f.DB.Exec(`INSERT INTO user_feature_grant(user_id,feature_key,granted_by_user_id) VALUES($1,'receipt_ocr',$1)`, f.User)
	require.NoError(t, err)
	response := request(t, s, f.User, "PATCH", "/account/ocr-improvement", `{"enabled":true,"policyVersion":"ocr-improvement-v1","expectedRevision":0}`)
	require.Equal(t, 200, response.Code)
	revision, err := user.NewStore(f.DB).GetOCRConsentRevision(t.Context(), f.User.String())
	require.NoError(t, err)
	require.Zero(t, revision)
	response = request(t, s, f.User, "PATCH", "/account/ocr-improvement", `{"enabled":false,"policyVersion":"ocr-improvement-v1","expectedRevision":2}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	p, err := s.Preference(t.Context(), f.User.String())
	require.NoError(t, err)
	require.False(t, p.Enabled)
	require.False(t, p.PurgePending)
}
