// Shared spec model for the generator: parses openapi.json into resource
// groups with the final SDK method names and signatures.
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"sort"
	"strings"
)

var httpMethods = []string{"get", "post", "put", "patch", "delete"}

// tagKeys maps OpenAPI tags to exported resource names on the Client.
// New/unknown tags fall back to auto-PascalCase.
var tagKeys = map[string]string{
	"Articles":          "Articles",
	"Images":            "Images",
	"Videos":            "Videos",
	"Social":            "Social",
	"Content":           "Content",
	"SEO":               "SEO",
	"Audio":             "Audio",
	"URLs":              "URLs",
	"AI Captain":        "Captain",
	"Chatbot":           "Chatbots",
	"Media":             "Media",
	"Analytics":         "Analytics",
	"Jobs":              "Jobs",
	"Account":           "Account",
	"Comments":          "Comments",
	"Direct Messages":   "Messages",
	"Webhooks":          "Webhooks",
	"Shorts":            "Shorts",
	"Presentations":     "Presentations",
	"CRM Contacts":      "Contacts",
	"CRM Opportunities": "Opportunities",
	"Workspaces":        "Workspaces",
	"CRM Custom Fields": "CustomFields",
	"Profiles":          "Profiles",
}

// extraStopwords lists extra noise words stripped from method names, per tag.
var extraStopwords = map[string][]string{
	"AI Captain":        {"ai"},
	"Direct Messages":   {"direct"},
	"CRM Contacts":      {"crm"},
	"CRM Opportunities": {"crm"},
	"CRM Custom Fields": {"crm"},
}

type pathParam struct {
	Arg string // Go argument name (camelCase)
	Raw string // raw spec name, e.g. article_id
}

type methodSpec struct {
	Name        string
	OperationID string
	HTTPMethod  string
	Path        string
	Summary     string
	PathParams  []pathParam
	HasBody     bool
	HasQuery    bool
}

type resourceSpec struct {
	Tag        string
	Field      string // exported field on Client, e.g. Social
	StructName string // e.g. SocialResource
	Methods    []*methodSpec
}

type specParam struct {
	Ref  string `json:"$ref"`
	Name string `json:"name"`
	In   string `json:"in"`
}

type specOperation struct {
	OperationID string      `json:"operationId"`
	Summary     string      `json:"summary"`
	Tags        []string    `json:"tags"`
	Parameters  []specParam `json:"parameters"`
	RequestBody *struct{}   `json:"requestBody"`
}

type specDoc struct {
	Paths      json.RawMessage `json:"paths"`
	Components struct {
		Parameters map[string]specParam `json:"parameters"`
	} `json:"components"`
}

// camelTokens splits an operationId into camelCase tokens, keeping acronym
// runs together (equivalent to /[A-Z]?[a-z0-9]+|[A-Z]+(?![a-z])/g).
func camelTokens(id string) []string {
	isUpper := func(r byte) bool { return r >= 'A' && r <= 'Z' }
	isLowerDigit := func(r byte) bool { return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') }
	var tokens []string
	i := 0
	for i < len(id) {
		switch {
		case isUpper(id[i]):
			j := i + 1
			for j < len(id) && isUpper(id[j]) {
				j++
			}
			if j-i > 1 && j < len(id) && isLowerDigit(id[j]) {
				// Acronym run followed by a word: last upper starts the next token.
				tokens = append(tokens, id[i:j-1])
				i = j - 1
			} else if j-i == 1 {
				k := j
				for k < len(id) && isLowerDigit(id[k]) {
					k++
				}
				tokens = append(tokens, id[i:k])
				i = k
			} else {
				tokens = append(tokens, id[i:j])
				i = j
			}
		case isLowerDigit(id[i]):
			k := i
			for k < len(id) && isLowerDigit(id[k]) {
				k++
			}
			tokens = append(tokens, id[i:k])
			i = k
		default:
			i++
		}
	}
	if len(tokens) == 0 {
		return []string{id}
	}
	return tokens
}

func pascal(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func snakeToCamel(name string) string {
	parts := strings.Split(name, "_")
	out := parts[0]
	for _, p := range parts[1:] {
		out += pascal(p)
	}
	return out
}

// stopwordsFor returns the tag's words plus singular/plural variants, all of
// which are stripped from operationIds when deriving method names.
func stopwordsFor(tag string) map[string]bool {
	words := strings.Fields(strings.ToLower(tag))
	for _, extra := range extraStopwords[tag] {
		words = append(words, strings.ToLower(extra))
	}
	set := map[string]bool{}
	for _, w := range words {
		set[w] = true
		switch {
		case strings.HasSuffix(w, "ies"):
			set[w[:len(w)-3]+"y"] = true
		case strings.HasSuffix(w, "s"):
			set[w[:len(w)-1]] = true
		default:
			set[w+"s"] = true
			if strings.HasSuffix(w, "y") {
				set[w[:len(w)-1]+"ies"] = true
			}
		}
	}
	return set
}

// methodName strips tag stopwords from the operationId and PascalCases the
// rest; if nothing is left it falls back to the full operationId.
func methodName(tag, operationID string) string {
	stop := stopwordsFor(tag)
	var kept []string
	for _, t := range camelTokens(operationID) {
		if !stop[strings.ToLower(t)] {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return pascal(operationID)
	}
	var b strings.Builder
	for _, t := range kept {
		b.WriteString(pascal(t))
	}
	return b.String()
}

// orderedPaths returns the spec's path keys in file order plus a lookup map,
// preserving the spec's ordering (encoding/json maps do not).
func orderedPaths(raw json.RawMessage) ([]string, map[string]map[string]json.RawMessage) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil { // opening {
		log.Fatalf("parsing paths: %v", err)
	}
	var keys []string
	byPath := map[string]map[string]json.RawMessage{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			log.Fatalf("parsing paths: %v", err)
		}
		key := tok.(string)
		var item map[string]json.RawMessage
		if err := dec.Decode(&item); err != nil {
			log.Fatalf("parsing path %s: %v", key, err)
		}
		keys = append(keys, key)
		byPath[key] = item
	}
	return keys, byPath
}

func buildModel(specPath string) []*resourceSpec {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		log.Fatalf("reading %s: %v", specPath, err)
	}
	var doc specDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		log.Fatalf("parsing %s: %v", specPath, err)
	}

	resolveParam := func(p specParam) specParam {
		if p.Ref != "" {
			parts := strings.Split(p.Ref, "/")
			return doc.Components.Parameters[parts[len(parts)-1]]
		}
		return p
	}

	pathKeys, byPath := orderedPaths(doc.Paths)
	byTag := map[string][]*methodSpec{}
	var tagOrder []string

	for _, path := range pathKeys {
		for _, httpMethod := range httpMethods {
			rawOp, ok := byPath[path][httpMethod]
			if !ok {
				continue
			}
			var op specOperation
			if err := json.Unmarshal(rawOp, &op); err != nil {
				log.Fatalf("parsing %s %s: %v", httpMethod, path, err)
			}
			tag := "Other"
			if len(op.Tags) > 0 {
				tag = op.Tags[0]
			}
			summary := op.Summary
			if summary == "" {
				summary = strings.ToUpper(httpMethod) + " " + path
			}
			m := &methodSpec{
				Name:        methodName(tag, op.OperationID),
				OperationID: op.OperationID,
				HTTPMethod:  strings.ToUpper(httpMethod),
				Path:        path,
				Summary:     summary,
				HasBody:     op.RequestBody != nil,
			}
			for _, p := range op.Parameters {
				rp := resolveParam(p)
				switch rp.In {
				case "path":
					m.PathParams = append(m.PathParams, pathParam{Arg: snakeToCamel(rp.Name), Raw: rp.Name})
				case "query":
					m.HasQuery = true
				}
			}
			if _, seen := byTag[tag]; !seen {
				tagOrder = append(tagOrder, tag)
			}
			byTag[tag] = append(byTag[tag], m)
		}
	}

	// Collision guard: if two ops in a resource shorten to the same name,
	// keep the full operationIds.
	for _, methods := range byTag {
		counts := map[string]int{}
		for _, m := range methods {
			counts[m.Name]++
		}
		for _, m := range methods {
			if counts[m.Name] > 1 {
				m.Name = pascal(m.OperationID)
			}
		}
	}

	sort.Slice(tagOrder, func(i, j int) bool {
		return strings.ToLower(tagOrder[i]) < strings.ToLower(tagOrder[j])
	})

	var resources []*resourceSpec
	for _, tag := range tagOrder {
		field, ok := tagKeys[tag]
		if !ok {
			var b strings.Builder
			for _, t := range camelTokens(strings.ReplaceAll(tag, " ", "")) {
				b.WriteString(pascal(t))
			}
			field = b.String()
		}
		resources = append(resources, &resourceSpec{
			Tag:        tag,
			Field:      field,
			StructName: field + "Resource",
			Methods:    byTag[tag],
		})
	}
	return resources
}
