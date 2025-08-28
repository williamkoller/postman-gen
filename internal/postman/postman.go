package postman

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/williamkoller/postman-gen/internal/scan"
)

const schemaV21 = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

type BuildOpts struct {
	Name          string
	BaseURL       string
	GroupDepth    int  // 0 = plano
	GroupByMethod bool // cria subpastas GET/POST/...
	TagFolders    bool // cria árvore "By Tag"
}

type Collection struct {
	Info     Info       `json:"info"`
	Item     []Item     `json:"item"`
	Variable []Variable `json:"variable,omitempty"`
}

type Info struct {
	Name        string `json:"name"`
	PostmanID   string `json:"_postman_id"`
	Schema      string `json:"schema"`
	Description string `json:"description,omitempty"`
}

type Item struct {
	Name     string   `json:"name"`
	Request  *Request `json:"request,omitempty"`
	Response []any    `json:"response,omitempty"`
	Item     []Item   `json:"item,omitempty"`
}

type Request struct {
	Method      string   `json:"method"`
	Header      []Header `json:"header"`
	Body        *Body    `json:"body,omitempty"`
	URL         URL      `json:"url"`
	Description string   `json:"description,omitempty"`
}

type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type Body struct {
	Mode    string                 `json:"mode"`
	Raw     string                 `json:"raw,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type URL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host"`
	Path  []string `json:"path"`
	Query []Query  `json:"query,omitempty"`
}

type Query struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
}

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

func BuildCollection(opts BuildOpts, eps []scan.Endpoint) Collection {
	if opts.GroupDepth < 0 {
		opts.GroupDepth = 0
	}
	sort.SliceStable(eps, func(i, j int) bool {
		if eps[i].Path == eps[j].Path {
			if eps[i].Method == eps[j].Method {
				return eps[i].SourceFile < eps[j].SourceFile
			}
			return eps[i].Method < eps[j].Method
		}
		return eps[i].Path < eps[j].Path
	})

	var mainTree []Item
	if opts.GroupDepth == 0 {
		for _, e := range eps {
			leaf := buildLeafItem(opts.BaseURL, e)
			if opts.GroupByMethod {
				insertMethodFolder(&mainTree, e.Method, leaf)
			} else {
				mainTree = append(mainTree, leaf)
			}
		}
	} else {
		for _, e := range eps {
			segments := splitPath(e.Path)
			group := take(segments, opts.GroupDepth)
			leaf := buildLeafItem(opts.BaseURL, e)
			if opts.GroupByMethod {
				insertIntoFolders(&mainTree, group, Item{Name: strings.ToUpper(e.Method), Item: []Item{leaf}}, true)
			} else {
				insertIntoFolders(&mainTree, group, leaf, false)
			}
		}
		normalizeMethodFolders(&mainTree)
	}

	if opts.TagFolders {
		byTag := buildTagTree(opts.BaseURL, eps)
		if len(byTag) > 0 {
			mainTree = append(mainTree, Item{Name: "By Tag", Item: byTag})
		}
	}

	return Collection{
		Info: Info{
			Name:      opts.Name,
			PostmanID: uuidV4(),
			Schema:    schemaV21,
		},
		Item: mainTree,
		Variable: []Variable{
			{Key: "baseUrl", Value: opts.BaseURL, Type: "string"},
		},
	}
}

func buildLeafItem(baseURL string, e scan.Endpoint) Item {
	title := strings.TrimSpace(strings.ToUpper(e.Method) + " " + e.Path)
	req := endpointToRequest(e)
	return Item{Name: title, Request: &req, Response: []any{}}
}

func pathToURL(path string) URL {
	raw := "{{baseUrl}}" + cleanPath(path)
	host := []string{"{{baseUrl}}"}
	pathSegments := splitPath(path)

	return URL{
		Raw:  raw,
		Host: host,
		Path: pathSegments,
	}
}

func endpointToRequest(e scan.Endpoint) Request {
	headers := []Header{}
	for k, v := range e.Headers {
		headers = append(headers, Header{Key: k, Value: v})
	}

	// Set default Content-Type based on endpoint type
	hasContentType := false
	for _, h := range headers {
		if strings.ToLower(h.Key) == "content-type" {
			hasContentType = true
			break
		}
	}

	var body *Body
	if e.Type == "GraphQL" {
		// GraphQL specific setup
		if !hasContentType {
			headers = append(headers, Header{Key: "Content-Type", Value: "application/json"})
		}

		// Build GraphQL body
		graphqlBody := map[string]interface{}{}
		if e.GraphQL != nil && e.GraphQL.Query != "" {
			graphqlBody["query"] = e.GraphQL.Query
		} else {
			// Default GraphQL query based on operation type
			if e.GraphQL != nil {
				switch e.GraphQL.Operation {
				case "mutation":
					graphqlBody["query"] = "mutation { # Add your mutation here }"
				case "subscription":
					graphqlBody["query"] = "subscription { # Add your subscription here }"
				default:
					graphqlBody["query"] = "query { # Add your query here }"
				}
			} else {
				graphqlBody["query"] = "query { # Add your query here }"
			}
		}

		if e.GraphQL != nil && e.GraphQL.Variables != "" {
			graphqlBody["variables"] = e.GraphQL.Variables
		}

		bodyJSON, _ := json.Marshal(graphqlBody)
		body = &Body{
			Mode: "raw",
			Raw:  string(bodyJSON),
			Options: map[string]interface{}{
				"raw": map[string]interface{}{
					"language": "json",
				},
			},
		}
	} else if e.BodyRaw != "" {
		// REST or other types with body
		if !hasContentType {
			headers = append(headers, Header{Key: "Content-Type", Value: "application/json"})
		}
		body = &Body{
			Mode: "raw",
			Raw:  e.BodyRaw,
			Options: map[string]interface{}{
				"raw": map[string]interface{}{
					"language": "json",
				},
			},
		}
	}

	desc := e.Desc
	if desc == "" {
		desc = "Source: " + e.SourceFile
		if e.Handler != "" {
			desc += " | Handler: " + e.Handler
		}
		if e.Type != "" {
			desc += " | Type: " + e.Type
		}
		if e.Type == "GraphQL" && e.GraphQL != nil && e.GraphQL.Operation != "" {
			desc += " | Operation: " + e.GraphQL.Operation
		}
	}

	return Request{
		Method:      e.Method,
		Header:      headers,
		Body:        body,
		URL:         pathToURL(e.Path),
		Description: desc,
	}
}

func insertMethodFolder(root *[]Item, method string, leaf Item) {
	method = strings.ToUpper(method)
	for i := range *root {
		if (*root)[i].Request == nil && (*root)[i].Name == method {
			(*root)[i].Item = append((*root)[i].Item, leaf)
			return
		}
	}
	*root = append(*root, Item{Name: method, Item: []Item{leaf}})
}

func insertIntoFolders(root *[]Item, group []string, child Item, childIsMethodFolder bool) {
	if len(group) == 0 || (len(group) == 1 && group[0] == "") {
		if childIsMethodFolder {
			var existing *Item
			for i := range *root {
				if (*root)[i].Request == nil && (*root)[i].Name == child.Name {
					existing = &(*root)[i]
					break
				}
			}
			if existing == nil {
				*root = append(*root, child)
			} else {
				existing.Item = append(existing.Item, child.Item...)
			}
		} else {
			*root = append(*root, child)
		}
		return
	}
	head := group[0]
	var folder *Item
	for i := range *root {
		if (*root)[i].Request == nil && (*root)[i].Name == head {
			folder = &(*root)[i]
			break
		}
	}
	if folder == nil {
		newFolder := Item{Name: head, Item: []Item{}}
		*root = append(*root, newFolder)
		folder = &(*root)[len(*root)-1]
	}
	if len(group) == 1 {
		if childIsMethodFolder {
			var mf *Item
			for i := range folder.Item {
				if folder.Item[i].Request == nil && folder.Item[i].Name == child.Name {
					mf = &folder.Item[i]
					break
				}
			}
			if mf == nil {
				folder.Item = append(folder.Item, child)
			} else {
				mf.Item = append(mf.Item, child.Item...)
			}
		} else {
			folder.Item = append(folder.Item, child)
		}
	} else {
		insertIntoFolders(&folder.Item, group[1:], child, childIsMethodFolder)
	}
}

func normalizeMethodFolders(nodes *[]Item) {
	for i := range *nodes {
		if (*nodes)[i].Request == nil {
			normalizeMethodFolders(&(*nodes)[i].Item)
		}
	}
}

func buildTagTree(baseURL string, eps []scan.Endpoint) []Item {
	buckets := map[string][]Item{}
	for _, e := range eps {
		if len(e.Tags) == 0 {
			continue
		}
		leaf := buildLeafItem(baseURL, e)
		for _, t := range e.Tags {
			tag := strings.TrimSpace(t)
			if tag == "" {
				continue
			}
			buckets[tag] = append(buckets[tag], leaf)
		}
	}
	if len(buckets) == 0 {
		return nil
	}
	tags := make([]string, 0, len(buckets))
	for t := range buckets {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	out := make([]Item, 0, len(tags))
	for _, t := range tags {
		out = append(out, Item{Name: t, Item: buckets[t]})
	}
	return out
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func splitPath(p string) []string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return []string{""}
	}
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func take(parts []string, n int) []string {
	if n <= 0 || len(parts) == 0 {
		return nil
	}
	if n > len(parts) {
		n = len(parts)
	}
	return append([]string(nil), parts[:n]...)
}

func uuidV4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexs := hex.EncodeToString(b[:])
	return hexs[0:8] + "-" + hexs[8:12] + "-" + hexs[12:16] + "-" + hexs[16:20] + "-" + hexs[20:]
}
