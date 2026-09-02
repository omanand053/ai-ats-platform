package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"ai-ats-platform/backend/internal/domain"

	"github.com/ledongthuc/pdf"
)

var (
	emailRegex = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phoneRegex = regexp.MustCompile(`(?:\+?\d{1,3}[\s\-.]?)?(?:\(?\d{2,4}\)?[\s\-.]?)?\d{3,4}[\s\-.]?\d{3,4}`)
	yearsRegex = regexp.MustCompile(`(?i)(\d{1,2}(?:\.\d+)?)\s*\+?\s*(?:years?|yrs?)(?:\s+of)?\s*(?:experience|exp\.?)?`)
)

var educationKeywords = []string{
	"university", "college", "school", "institute", "bachelor", "master", "b.tech", "b.e",
	"m.tech", "mba", "bsc", "msc", "phd", "polytechnic", "academy", "education",
}

var skillCanonical = map[string]string{
	"javascript": "JavaScript", "java script": "JavaScript", "js": "JavaScript", "ecmascript": "JavaScript",
	"typescript": "TypeScript", "ts": "TypeScript",
	"react": "React", "react.js": "React", "reactjs": "React", "react js": "React",
	"node": "Node.js", "nodejs": "Node.js", "node.js": "Node.js", "node js": "Node.js",
	"next": "Next.js", "nextjs": "Next.js", "next.js": "Next.js", "next js": "Next.js",
	"mongodb": "MongoDB", "mongo": "MongoDB", "mongo db": "MongoDB",
	"postgresql": "PostgreSQL", "postgres": "PostgreSQL", "psql": "PostgreSQL",
	"mysql": "MySQL",
	"golang": "Go", "go": "Go",
	"python": "Python", "py": "Python",
	"java": "Java",
	"docker": "Docker",
	"kubernetes": "Kubernetes", "k8s": "Kubernetes",
	"aws": "AWS", "amazon web services": "AWS",
	"azure": "Azure",
	"gcp": "GCP", "google cloud": "GCP",
	"html": "HTML", "css": "CSS",
	"tailwind": "Tailwind", "tailwindcss": "Tailwind",
	"vue": "Vue", "vue.js": "Vue", "vuejs": "Vue",
	"angular": "Angular",
	"redux": "Redux",
	"graphql": "GraphQL",
	"rest": "REST",
	"git": "Git",
	"linux": "Linux",
	"redis": "Redis",
	"kafka": "Kafka",
	"terraform": "Terraform",
	"ci/cd": "CI/CD", "cicd": "CI/CD",
	"c++": "C++", "cpp": "C++",
	"c#": "C#", "csharp": "C#",
	".net": ".NET", "dotnet": ".NET",
	"ruby": "Ruby", "rails": "Rails", "ruby on rails": "Rails",
	"php": "PHP", "laravel": "Laravel",
	"swift": "Swift", "kotlin": "Kotlin", "flutter": "Flutter",
	"tensorflow": "TensorFlow", "pytorch": "PyTorch", "pandas": "Pandas",
	"spark": "Spark", "elasticsearch": "Elasticsearch",
	"express": "Express", "express.js": "Express", "expressjs": "Express",
}

var skillSeedList = []string{
	"Go", "Python", "Java", "JavaScript", "TypeScript", "React", "Next.js", "Node.js",
	"PostgreSQL", "MySQL", "MongoDB", "Redis", "Docker", "Kubernetes", "AWS", "Azure", "GCP",
	"Git", "GraphQL", "REST", "Linux", "HTML", "CSS", "Tailwind", "Vue", "Angular",
	"C++", "C#", ".NET", "Ruby", "Rails", "PHP", "Laravel", "Swift", "Kotlin", "Flutter",
	"TensorFlow", "PyTorch", "Pandas", "Spark", "Kafka", "Elasticsearch", "Terraform", "CI/CD",
	"Express", "Redux",
}

// ExtractTextFromResume returns plain text for PDF or DOCX bytes.
func ExtractTextFromResume(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return extractPDFText(data)
	case ".docx":
		return extractDOCXText(data)
	case ".txt", ".text", ".md":
		return preserveReadableText(string(data)), nil
	default:
		return "", fmt.Errorf("unsupported file type %s", ext)
	}
}

func extractPDFText(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("read pdf: %w", err)
	}

	var b strings.Builder
	total := reader.NumPage()
	for i := 1; i <= total; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	return preserveReadableText(b.String()), nil
}

func extractDOCXText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("read docx: %w", err)
	}

	var xmlContent string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			raw, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return "", err
			}
			xmlContent = string(raw)
			break
		}
	}
	if xmlContent == "" {
		return "", fmt.Errorf("docx missing document.xml")
	}

	xmlContent = strings.ReplaceAll(xmlContent, "</w:p>", "\n")
	xmlContent = strings.ReplaceAll(xmlContent, "<w:tab/>", "\t")
	xmlContent = strings.ReplaceAll(xmlContent, "<w:br/>", "\n")
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(xmlContent, "")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	return preserveReadableText(text), nil
}

// ParseResumeText extracts structured candidate fields from resume plain text.
func ParseResumeText(text string) domain.ParsedResume {
	raw := preserveReadableText(text)
	lines := nonEmptyLines(raw)

	parsed := domain.ParsedResume{
		Skills:         []string{},
		Projects:       []domain.ParsedProject{},
		Education:      []domain.ParsedEducation{},
		Certifications: []string{},
		RawText:        raw,
	}

	if email := emailRegex.FindString(raw); email != "" {
		parsed.Email = strings.ToLower(email)
	}
	if phone := findPhone(raw); phone != "" {
		parsed.Phone = phone
	}

	parsed.FullName = guessName(lines, parsed.Email)
	parsed.Location = extractLocation(lines, raw)
	parsed.ExperienceYears = extractExperienceYears(raw)
	parsed.Education = extractEducation(raw)
	parsed.Certifications = extractCertifications(raw)
	parsed.Skills = extractSkillsNormalized(raw)
	parsed.Projects = extractProjects(raw)
	parsed.CurrentCompany, parsed.CurrentDesignation = extractCurrentRole(raw, lines)
	parsed.Summary = extractSummaryClean(raw, lines)

	return parsed
}

func preserveReadableText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Collapse spaces/tabs on a line but keep newlines.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = strings.ReplaceAll(line, "\t", " ")
		line = regexp.MustCompile(` {2,}`).ReplaceAllString(line, " ")
		lines[i] = strings.TrimRight(line, " ")
	}
	s = strings.Join(lines, "\n")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func nonEmptyLines(text string) []string {
	raw := strings.Split(text, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func findPhone(text string) string {
	matches := phoneRegex.FindAllString(text, -1)
	for _, m := range matches {
		digits := onlyDigits(m)
		if len(digits) >= 7 && len(digits) <= 15 {
			return strings.TrimSpace(m)
		}
	}
	return ""
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func guessName(lines []string, email string) string {
	for _, line := range lines[:min(10, len(lines))] {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "@") || strings.Contains(lower, "http") {
			continue
		}
		if strings.Contains(lower, "curriculum") || strings.Contains(lower, "resume") {
			continue
		}
		if phoneRegex.MatchString(line) && len(onlyDigits(line)) >= 7 {
			continue
		}
		if isEducationLine(line) {
			continue
		}
		words := strings.Fields(line)
		if len(words) < 2 || len(words) > 5 {
			continue
		}
		if looksLikeName(line) {
			return titleCaseName(line)
		}
	}

	if email != "" {
		local := strings.Split(email, "@")[0]
		local = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(local)
		parts := strings.Fields(local)
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if len(p) < 2 || isNumeric(p) {
				continue
			}
			cleaned = append(cleaned, strings.ToUpper(p[:1])+strings.ToLower(p[1:]))
		}
		if len(cleaned) >= 2 {
			return strings.Join(cleaned, " ")
		}
	}
	return ""
}

func titleCaseName(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == strings.ToUpper(p) && len(p) > 1 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

func looksLikeName(line string) bool {
	for _, r := range line {
		if unicode.IsDigit(r) {
			return false
		}
	}
	words := strings.Fields(line)
	for _, w := range words {
		if len(w) < 2 {
			return false
		}
		runes := []rune(w)
		if !unicode.IsLetter(runes[0]) {
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

func extractLocation(lines []string, text string) string {
	for _, line := range nonEmptyLines(text) {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "location:") || strings.HasPrefix(lower, "based in:") ||
			strings.HasPrefix(lower, "address:") {
			val := strings.TrimSpace(line[strings.Index(line, ":")+1:])
			if loc := cleanLocationValue(val); loc != "" {
				return loc
			}
		}
	}
	for _, line := range lines[:min(8, len(lines))] {
		if loc := confidentHeaderLocation(line); loc != "" {
			return loc
		}
	}
	return ""
}

func confidentHeaderLocation(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.Contains(line, "@") || isEducationLine(line) || isSectionHeading(line) || looksLikeRole(line) {
		return ""
	}
	if phoneRegex.MatchString(line) && len(onlyDigits(line)) >= 7 && !strings.Contains(line, ",") {
		return ""
	}
	lower := strings.ToLower(line)
	if lower == "remote" || lower == "work from home" {
		return "Remote"
	}
	known := []string{
		"bangalore", "bengaluru", "mumbai", "delhi", "new delhi", "hyderabad", "chennai", "pune", "kolkata",
		"noida", "gurgaon", "gurugram", "ahmedabad", "jaipur", "remote",
		"san francisco", "new york", "seattle", "austin", "london", "berlin", "toronto", "singapore",
		"india", "usa", "uk", "united states", "united kingdom", "canada", "germany", "australia",
		"california", "texas", "washington", "karnataka", "maharashtra", "tamil nadu", "telangana",
	}
	compact := strings.ToLower(strings.TrimSpace(line))
	for _, city := range known {
		if compact == city || strings.HasPrefix(compact, city+",") {
			return cleanLocationValue(line)
		}
	}
	cityRe := regexp.MustCompile(`(?i)^([A-Za-z][A-Za-z .']{1,30}),\s*([A-Za-z][A-Za-z .']{1,30})$`)
	if m := cityRe.FindStringSubmatch(line); len(m) == 3 {
		left, right := strings.ToLower(strings.TrimSpace(m[1])), strings.ToLower(strings.TrimSpace(m[2]))
		if isEducationLine(m[0]) {
			return ""
		}
		for _, city := range known {
			if left == city || right == city {
				return cleanLocationValue(m[0])
			}
		}
	}
	return ""
}

func cleanLocationValue(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "|•-")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	if len(s) < 3 || len(s) > 80 {
		return ""
	}
	if isEducationLine(s) || looksLikeRole(s) {
		return ""
	}
	return s
}

func extractLabeledCompanyRole(text string) (company, role string) {
	for _, line := range nonEmptyLines(text) {
		lower := strings.ToLower(strings.TrimSpace(line))
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		if val == "" {
			continue
		}
		switch {
		case strings.HasPrefix(lower, "current company:") || strings.HasPrefix(lower, "company:") ||
			strings.HasPrefix(lower, "employer:"):
			if company == "" && !isEducationLine(val) {
				company = val
			}
		case strings.HasPrefix(lower, "current designation:") || strings.HasPrefix(lower, "designation:") ||
			strings.HasPrefix(lower, "position:") || strings.HasPrefix(lower, "job title:") ||
			strings.HasPrefix(lower, "title:"):
			if role == "" {
				role = val
			}
		}
	}
	return company, role
}

// extractCurrentRole prefers labeled fields, then the first clear role in Experience.
func extractCurrentRole(text string, lines []string) (company, role string) {
	if c, r := extractLabeledCompanyRole(text); c != "" || r != "" {
		return c, r
	}

	exp := sectionBody(text, []string{
		"experience", "work experience", "employment", "professional experience", "work history",
	})
	srcLines := nonEmptyLines(exp)
	if len(srcLines) == 0 {
		body := sectionBody(text, []string{"summary", "profile", "professional summary", "about me"})
		if body != "" {
			for _, line := range nonEmptyLines(body) {
				if roleFromSummaryLine(line) != "" {
					return "", roleFromSummaryLine(line)
				}
			}
		}
		return "", ""
	}

	for i, line := range srcLines {
		if isEducationLine(line) || isSectionHeading(line) {
			continue
		}
		c, r := parseRoleLine(line)
		if c != "" && isEducationLine(c) {
			continue
		}
		if c == "" && r == "" && i+1 < len(srcLines) && !isEducationLine(srcLines[i+1]) {
			if looksLikeRole(line) && !looksLikeRole(srcLines[i+1]) {
				r = strings.TrimSpace(line)
				c = strings.TrimSpace(srcLines[i+1])
			} else if !looksLikeRole(line) && looksLikeRole(srcLines[i+1]) {
				c = strings.TrimSpace(line)
				r = strings.TrimSpace(srcLines[i+1])
			}
		}
		if (c != "" || r != "") && !isEducationLine(c) {
			// Prefer pairs where role looks like a job title.
			if r != "" && !looksLikeRole(r) && c != "" && looksLikeRole(c) {
				c, r = r, c
			}
			if r != "" && !looksLikeRole(r) {
				continue
			}
			return strings.TrimSpace(c), strings.TrimSpace(r)
		}
	}
	_ = lines
	return "", ""
}

func parseRoleLine(line string) (company, role string) {
	lower := strings.ToLower(line)
	if idx := strings.Index(lower, " at "); idx >= 0 {
		role = strings.TrimSpace(line[:idx])
		company = strings.TrimSpace(line[idx+4:])
		return company, role
	}
	if role := roleFromSummaryLine(line); role != "" {
		return "", role
	}
	for _, sep := range []string{" - ", " – ", " — ", " | ", " @ "} {
		if strings.Contains(line, sep) {
			parts := strings.SplitN(line, sep, 2)
			left, right := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if looksLikeRole(left) && !looksLikeRole(right) {
				return right, left
			}
			if looksLikeRole(right) && !looksLikeRole(left) {
				return left, right
			}
			if looksLikeRole(left) {
				return right, left
			}
			if looksLikeRole(right) {
				return left, right
			}
		}
	}
	return "", ""
}

func roleFromSummaryLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"full stack developer", "front end developer", "frontend developer", "backend developer", "software engineer", "data engineer", "data scientist", "product manager", "qa engineer", "devops engineer", "ai engineer", "ml engineer", "cloud engineer", "sre", "recruiter", "hr", "business development", "marketing", "sales"} {
		if strings.Contains(lower, prefix) {
			return trimmed
		}
	}
	return ""
}

func extractCertifications(text string) []string {
	body := sectionBody(text, []string{
		"certifications", "certificates", "licenses", "professional certifications",
	})
	if body == "" {
		return []string{}
	}
	items := []string{}
	seen := map[string]bool{}
	for _, line := range nonEmptyLines(body) {
		line = strings.TrimSpace(strings.TrimLeft(line, "-•* \t"))
		if len(line) < 3 || isSectionHeading(line) {
			continue
		}
		key := strings.ToLower(line)
		if seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, line)
		if len(items) >= 10 {
			break
		}
	}
	return items
}

func extractExperienceYears(text string) *int {
	// Prefer explicit "X years of experience" outside education sections.
	workText := stripSection(text, []string{"education", "academic", "qualifications"})
	best := -1.0

	strong := regexp.MustCompile(`(?i)(\d{1,2}(?:\.\d+)?)\s*\+?\s*(?:years?|yrs?)\s+(?:of\s+)?(?:experience|exp\b)`)
	for _, m := range strong.FindAllStringSubmatch(workText, -1) {
		if len(m) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil || v > 50 {
			continue
		}
		if v > best {
			best = v
		}
	}

	if best < 0 {
		for _, m := range yearsRegex.FindAllStringSubmatch(workText, -1) {
			if len(m) < 2 {
				continue
			}
			idx := strings.Index(strings.ToLower(workText), strings.ToLower(m[0]))
			if idx >= 0 {
				start := max(0, idx-80)
				end := min(len(workText), idx+len(m[0])+80)
				window := strings.ToLower(workText[start:end])
				if containsAny(window, educationKeywords) {
					continue
				}
			}
			v, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			if v > best && v <= 50 {
				best = v
			}
		}
	}
	if best < 0 {
		return nil
	}
	y := int(best)
	return &y
}

func extractEducation(text string) []domain.ParsedEducation {
	body := sectionBody(text, []string{"education", "academic", "qualifications", "academics"})
	if body == "" {
		return []domain.ParsedEducation{}
	}
	items := []domain.ParsedEducation{}
	for _, line := range nonEmptyLines(body) {
		line = strings.TrimLeft(line, "-•* \t")
		if len(line) < 3 || isSectionHeading(line) {
			continue
		}
		items = append(items, domain.ParsedEducation{School: line})
		if len(items) >= 8 {
			break
		}
	}
	return items
}

func NormalizeSkillKey(skill string) string {
	s := strings.ToLower(strings.TrimSpace(skill))
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// Strip trailing .js variants already handled via map keys.
	if canon, ok := skillCanonical[s]; ok {
		return strings.ToLower(canon)
	}
	s2 := strings.ReplaceAll(s, " ", "")
	s2 = strings.ReplaceAll(s2, ".", "")
	if canon, ok := skillCanonical[s2]; ok {
		return strings.ToLower(canon)
	}
	return s2
}

func CanonicalSkillName(skill string) string {
	key := NormalizeSkillKey(skill)
	for k, v := range skillCanonical {
		if NormalizeSkillKey(k) == key || strings.ToLower(v) == key {
			return v
		}
	}
	// Title-ish fallback
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return ""
	}
	return skill
}

func extractSkillsNormalized(text string) []string {
	found := make([]string, 0)
	seen := map[string]bool{}

	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		raw = strings.Trim(raw, "-•*|/")
		if len(raw) < 2 || len(raw) > 40 {
			return
		}
		canon := CanonicalSkillName(raw)
		if canon == "" {
			return
		}
		key := NormalizeSkillKey(canon)
		if seen[key] {
			return
		}
		seen[key] = true
		found = append(found, canon)
	}

	lower := strings.ToLower(text)
	hasJS := regexp.MustCompile(`(?i)\bjavascript\b|\bjava\s+script\b|\bjs\b`).MatchString(text)
	for _, seed := range skillSeedList {
		if strings.EqualFold(seed, "Java") && hasJS {
			continue
		}
		if regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(seed)+`\b`).MatchString(text) {
			add(seed)
		}
	}
	_ = lower
	// Also check aliases present in text.
	for alias, canon := range skillCanonical {
		if len(alias) < 2 {
			continue
		}
		if alias == "java" && hasJS {
			continue
		}
		if regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(alias)+`\b`).MatchString(text) {
			add(canon)
		}
	}

	section := sectionBody(text, []string{"skills", "technical skills", "technologies", "tech stack", "core competencies"})
	if section != "" {
		for _, part := range regexp.MustCompile(`[,|/•\n]`).Split(section, -1) {
			add(part)
			if len(found) >= 40 {
				break
			}
		}
	}
	return found
}

func extractProjects(text string) []domain.ParsedProject {
	body := sectionBody(text, []string{"projects", "project experience", "personal projects", "key projects"})
	if body == "" {
		return []domain.ParsedProject{}
	}

	projects := []domain.ParsedProject{}
	chunks := splitProjectChunks(body)
	for _, chunk := range chunks {
		lines := nonEmptyLines(chunk)
		if len(lines) == 0 {
			continue
		}
		name := strings.TrimLeft(lines[0], "-•* \t")
		if isSectionHeading(name) {
			continue
		}
		techs := []string{}
		if strings.Contains(name, "|") {
			parts := strings.SplitN(name, "|", 2)
			name = strings.TrimSpace(parts[0])
			for _, t := range regexp.MustCompile(`[,|/]`).Split(parts[1], -1) {
				t = strings.TrimSpace(t)
				if t != "" {
					techs = append(techs, CanonicalSkillName(t))
				}
			}
		}
		descParts := []string{}
		for _, line := range lines[1:] {
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "tech") || strings.HasPrefix(lower, "stack") || strings.Contains(lower, "technologies") {
				techLine := line
				if idx := strings.Index(line, ":"); idx >= 0 {
					techLine = line[idx+1:]
				}
				for _, t := range regexp.MustCompile(`[,|/]`).Split(techLine, -1) {
					t = strings.TrimSpace(t)
					if t != "" {
						techs = append(techs, CanonicalSkillName(t))
					}
				}
				continue
			}
			descParts = append(descParts, strings.TrimLeft(line, "-•* \t"))
		}
		// Detect inline techs in description.
		if len(techs) == 0 {
			for _, seed := range skillSeedList {
				if regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(seed)+`\b`).MatchString(chunk) {
					techs = append(techs, CanonicalSkillName(seed))
				}
			}
		}
		techs = dedupeSkills(techs)
		desc := strings.Join(descParts, " ")
		if len(desc) > 400 {
			desc = desc[:397] + "..."
		}
		projects = append(projects, domain.ParsedProject{
			Name:         name,
			Technologies: techs,
			Description:  desc,
		})
		if len(projects) >= 8 {
			break
		}
	}
	return projects
}

func splitProjectChunks(body string) []string {
	lines := strings.Split(body, "\n")
	chunks := []string{}
	var cur []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		chunks = append(chunks, strings.Join(cur, "\n"))
		cur = nil
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(cur) > 0 {
				flush()
			}
			continue
		}
		// New project often starts with Title Case / bullet heading without indent-like soft rules.
		if len(cur) > 0 && looksLikeProjectTitle(trimmed) {
			flush()
		}
		cur = append(cur, trimmed)
	}
	flush()
	if len(chunks) == 0 && strings.TrimSpace(body) != "" {
		return []string{body}
	}
	return chunks
}

func looksLikeProjectTitle(line string) bool {
	if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
		line = strings.TrimLeft(line, "-•* ")
	}
	words := strings.Fields(line)
	if len(words) == 0 || len(words) > 10 {
		return false
	}
	if len(line) > 80 {
		return false
	}
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "tech") || strings.HasPrefix(lower, "description") {
		return false
	}
	return true
}

func looksLikeRole(s string) bool {
	lower := strings.ToLower(s)
	keywords := []string{
		"engineer", "developer", "manager", "analyst", "designer", "lead", "intern",
		"consultant", "architect", "specialist", "administrator", "scientist", "founder",
		"director", "officer", "programmer", "sde", "swe",
	}
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func isEducationLine(s string) bool {
	return containsAny(strings.ToLower(s), educationKeywords)
}

func isSectionHeading(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	headings := []string{
		"experience", "education", "skills", "projects", "summary", "profile",
		"work experience", "technical skills", "certifications",
	}
	for _, h := range headings {
		if t == h || strings.HasPrefix(t, h+":") {
			return true
		}
	}
	return false
}

func extractSummaryClean(text string, lines []string) string {
	body := sectionBody(text, []string{
		"summary", "profile", "about me", "objective", "professional summary", "career summary",
	})
	if body != "" {
		parts := nonEmptyLines(body)
		if len(parts) > 6 {
			parts = parts[:6]
		}
		return strings.Join(parts, "\n")
	}

	// Fallback: short readable blurb from early paragraph lines (not contact/name).
	chunk := []string{}
	for i, line := range lines {
		if i == 0 && looksLikeName(line) {
			continue
		}
		if strings.Contains(line, "@") || (phoneRegex.MatchString(line) && len(onlyDigits(line)) >= 7) {
			continue
		}
		if isSectionHeading(line) {
			break
		}
		if len(line) > 35 {
			chunk = append(chunk, line)
		}
		if len(chunk) >= 3 {
			break
		}
	}
	return strings.Join(chunk, "\n")
}

func sectionBody(text string, headings []string) string {
	lines := strings.Split(text, "\n")
	lowerLines := make([]string, len(lines))
	for i, line := range lines {
		lowerLines[i] = strings.ToLower(strings.TrimSpace(line))
	}

	start := -1
	for i, line := range lowerLines {
		for _, h := range headings {
			if line == h || strings.HasPrefix(line, h+":") || line == strings.ToUpper(h) {
				start = i + 1
				break
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return ""
	}

	known := []string{
		"experience", "work experience", "employment", "education", "skills",
		"projects", "summary", "profile", "certifications", "achievements",
		"technical skills", "work history", "objective",
	}
	end := len(lines)
	for i := start; i < len(lowerLines); i++ {
		trimmed := lowerLines[i]
		for _, h := range known {
			skip := false
			for _, own := range headings {
				if h == own {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			if trimmed == h || strings.HasPrefix(trimmed, h+":") {
				end = i
				break
			}
		}
		if end != len(lines) {
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func stripSection(text string, headings []string) string {
	body := sectionBody(text, headings)
	if body == "" {
		return text
	}
	return strings.Replace(text, body, "\n", 1)
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func dedupeSkills(skills []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range skills {
		key := NormalizeSkillKey(s)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, CanonicalSkillName(s))
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
