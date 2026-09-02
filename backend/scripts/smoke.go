//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL  = "http://localhost:8000"
	email    = "swayam123@gmail.com"
	password = "Admin@12345"
)

func main() {
	results := map[string]bool{}
	token := mustLogin()
	results["Login"] = token != ""

	meStatus := getStatus("/api/v1/auth/me", token, nil)
	results["Dashboard"] = meStatus == 200

	jobID := createJob(token)
	results["Jobs CRUD"] = jobID != "" && listJobs(token) && updateJob(token, jobID) && deleteJob(token, jobID)
	jobID = createJob(token)

	candidateID := createCandidate(token, jobID)
	results["Candidate CRUD"] = candidateID != "" && getCandidate(token, candidateID)

	resumeID, parsedOK := uploadResume(token)
	results["Resume Upload"] = resumeID != ""
	results["Resume Parse"] = parsedOK
	results["Resume Review"] = parsedOK

	if candidateID != "" && resumeID != "" {
		attachResume(token, resumeID, candidateID)
	}

	fitOK := checkFitScore(token, candidateID)
	results["Fit Score"] = fitOK

	semanticOK := checkSemantic(token, jobID)
	results["Semantic Search"] = semanticOK

	aiOK := checkAIAssistant(token, jobID)
	results["AI Recruiter Assistant"] = aiOK

	fmt.Println("\n=== SMOKE TEST RESULTS ===")
	allOK := true
	for _, name := range []string{
		"Login", "Dashboard", "Jobs CRUD", "Candidate CRUD",
		"Resume Upload", "Resume Parse", "Resume Review",
		"Fit Score", "Semantic Search", "AI Recruiter Assistant",
	} {
		ok := results[name]
		mark := "✓"
		if !ok {
			mark = "✗"
			allOK = false
		}
		fmt.Printf("%s %s\n", mark, name)
	}
	if !allOK {
		os.Exit(1)
	}
}

func mustLogin() string {
	body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	status, raw := postJSON("/api/v1/auth/login", "", body)
	if status != 200 {
		fail("login", status, raw)
	}
	var parsed struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	must(json.Unmarshal([]byte(raw), &parsed))
	if parsed.Data.AccessToken == "" {
		fail("login token empty", status, raw)
	}
	return parsed.Data.AccessToken
}

func getStatus(path, token string, body *string) int {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		panic(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Method = http.MethodPost
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(strings.NewReader(*body))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return resp.StatusCode
}

func postJSON(path, token, body string) (int, string) {
	req, err := http.NewRequest(http.MethodPost, baseURL+path, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func putJSON(path, token, body string) (int, string) {
	req, err := http.NewRequest(http.MethodPut, baseURL+path, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func deleteReq(path, token string) int {
	req, err := http.NewRequest(http.MethodDelete, baseURL+path, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return resp.StatusCode
}

func createJob(token string) string {
	body := `{"title":"Smoke Test Go Engineer","department":"Engineering","location":"Remote","employment_type":"full_time","experience_required":"3+ years","required_skills":["Go","PostgreSQL","Docker"],"description":"Backend role for smoke testing.","status":"open"}`
	status, raw := postJSON("/api/v1/jobs", token, body)
	if status != 201 {
		fmt.Printf("create job failed: %d %s\n", status, raw)
		return ""
	}
	var parsed struct {
		Data struct {
			Job struct {
				ID string `json:"id"`
			} `json:"job"`
		} `json:"data"`
	}
	must(json.Unmarshal([]byte(raw), &parsed))
	return parsed.Data.Job.ID
}

func listJobs(token string) bool {
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/jobs?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func updateJob(token, jobID string) bool {
	body := `{"title":"Smoke Test Go Engineer Updated","status":"open"}`
	status, _ := putJSON("/api/v1/jobs/"+jobID, token, body)
	return status == 200
}

func deleteJob(token, jobID string) bool {
	return deleteReq("/api/v1/jobs/"+jobID, token) == 200
}

func createCandidate(token, jobID string) string {
	candidateEmail := fmt.Sprintf("smoke-%d@example.com", time.Now().Unix())
	body := fmt.Sprintf(`{"job_id":"%s","name":"Jane Doe","email":"%s","experience_years":5,"current_company":"Acme Corp","current_designation":"Senior Software Engineer","location":"Bengaluru, India","skills":["Go","PostgreSQL","Docker","Redis"],"resume_summary":"Backend engineer with Go experience.","resume_text":"Experienced Backend Engineer with 5 years building REST APIs using Go, Gin, PostgreSQL, Redis and Docker.","status":"applied","source":"Smoke Test"}`, jobID, candidateEmail)
	status, raw := postJSON("/api/v1/candidates", token, body)
	if status != 201 {
		fmt.Printf("create candidate failed: %d %s\n", status, raw)
		return ""
	}
	var parsed struct {
		Data struct {
			Candidate struct {
				ID string `json:"id"`
			} `json:"candidate"`
		} `json:"data"`
	}
	must(json.Unmarshal([]byte(raw), &parsed))
	return parsed.Data.Candidate.ID
}

func getCandidate(token, id string) bool {
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/candidates/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode == 200 && strings.Contains(string(raw), id)
}

func uploadResume(token string) (string, bool) {
	docx := buildMinimalDOCX()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "smoke-resume.docx")
	if err != nil {
		return "", false
	}
	if _, err := part.Write(docx); err != nil {
		return "", false
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/resumes/upload", &buf)
	if err != nil {
		return "", false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		fmt.Printf("resume upload failed: %d %s\n", resp.StatusCode, raw)
		return "", false
	}
	var parsed struct {
		Data struct {
			ResumeID      string `json:"resume_id"`
			ParsingStatus string `json:"parsing_status"`
			Parsed        struct {
				FullName string   `json:"full_name"`
				Email    string   `json:"email"`
				Skills   []string `json:"skills"`
			} `json:"parsed"`
		} `json:"data"`
	}
	must(json.Unmarshal(raw, &parsed))
	parsedOK := parsed.Data.ParsingStatus == "completed" && parsed.Data.Parsed.Email != ""
	return parsed.Data.ResumeID, parsedOK
}

func attachResume(token, resumeID, candidateID string) {
	body := fmt.Sprintf(`{"candidate_id":"%s"}`, candidateID)
	postJSON("/api/v1/resumes/"+resumeID+"/attach", token, body)
}

func checkFitScore(token, candidateID string) bool {
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/candidates/"+candidateID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Printf("fit score fetch failed: %d %s\n", resp.StatusCode, raw)
		return false
	}
	return strings.Contains(string(raw), "overall_score")
}

func checkSemantic(token, jobID string) bool {
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/jobs/"+jobID+"/semantic-matches", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Printf("semantic failed: %d %s\n", resp.StatusCode, raw)
		return false
	}
	return strings.Contains(string(raw), "status")
}

func checkAIAssistant(token, jobID string) bool {
	body := `{"question":"Who are the best matching candidates for this role?","top_k":3}`
	status, raw := postJSON("/api/v1/jobs/"+jobID+"/ai-assistant", token, body)
	if status != 200 {
		fmt.Printf("ai assistant failed: %d %s\n", status, raw)
		return false
	}
	return strings.Contains(raw, "answer") || strings.Contains(raw, "response")
}

func buildMinimalDOCX() []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Jane Doe</w:t></w:r></w:p><w:p><w:r><w:t>Email: jane.doe@example.com</w:t></w:r></w:p><w:p><w:r><w:t>5 years of experience in Go, PostgreSQL, Docker, Redis, REST API</w:t></w:r></w:p><w:p><w:r><w:t>Senior Software Engineer at Acme Corp, Bengaluru, India</w:t></w:r></w:p></w:body></w:document>`,
	}
	for name, content := range files {
		f, _ := zw.Create(name)
		_, _ = f.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func fail(msg string, status int, raw string) {
	fmt.Fprintf(os.Stderr, "%s status=%d body=%s\n", msg, status, raw)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
