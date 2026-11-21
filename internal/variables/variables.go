// Package variables implements Postman-compatible variable resolution,
// including dynamic variables such as {{$randomUUID}}.
package variables

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"strings"
	"sync"
	"time"
)

// Resolver resolves {{variable}} placeholders using ordered scopes. Earlier
// scopes win. Dynamic variables ({{$guid}}, {{$timestamp}}, ...) are generated
// lazily and cached for the lifetime of the resolver so a single request sees
// consistent values.
type Resolver struct {
	scopes  []map[string]string
	dynamic map[string]string
	mu      sync.Mutex
	rng     *mathrand.Rand
}

func New(scopes ...map[string]string) *Resolver {
	clean := make([]map[string]string, 0, len(scopes))
	for _, s := range scopes {
		if s != nil {
			clean = append(clean, s)
		}
	}
	return &Resolver{
		scopes:  clean,
		dynamic: map[string]string{},
		rng:     mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

// Set adds or overrides a value in the highest-priority scope.
func (r *Resolver) Set(key, value string) {
	if len(r.scopes) == 0 {
		r.scopes = append(r.scopes, map[string]string{})
	}
	r.scopes[0][key] = value
}

// Value looks up a variable by name (without braces).
func (r *Resolver) Value(key string) (string, bool) {
	if strings.HasPrefix(key, "$") {
		r.mu.Lock()
		v, ok := r.dynamic[key]
		r.mu.Unlock()
		if !ok {
			// generate() takes r.mu internally (via intn), so it must run
			// without the lock held to avoid a self-deadlock.
			v = r.generate(key)
			r.mu.Lock()
			r.dynamic[key] = v
			r.mu.Unlock()
		}
		return v, v != ""
	}
	for _, s := range r.scopes {
		if v, ok := s[key]; ok {
			return v, true
		}
	}
	return "", false
}

// Resolve replaces every {{name}} placeholder. Unresolved placeholders are left
// untouched, which is what Postman does and helps users spot typos.
func (r *Resolver) Resolve(input string) string {
	if input == "" || !strings.Contains(input, "{{") {
		return input
	}
	// Repeat to support nested variables, but cap iterations to avoid cycles.
	for i := 0; i < 10; i++ {
		changed := false
		var b strings.Builder
		b.Grow(len(input))
		for {
			start := strings.Index(input, "{{")
			if start < 0 {
				b.WriteString(input)
				break
			}
			end := strings.Index(input[start+2:], "}}")
			if end < 0 {
				b.WriteString(input)
				break
			}
			end += start + 2
			name := strings.TrimSpace(input[start+2 : end])
			if v, ok := r.Value(name); ok {
				b.WriteString(input[:start])
				b.WriteString(v)
				input = input[end+2:]
				changed = true
			} else {
				b.WriteString(input[:end+2])
				input = input[end+2:]
			}
		}
		result := b.String()
		input = result
		if !changed {
			break
		}
	}
	return input
}

// Object returns a flattened map of all static variables (highest priority
// wins), excluding dynamic variables.
func (r *Resolver) Object() map[string]string {
	out := map[string]string{}
	for i := len(r.scopes) - 1; i >= 0; i-- {
		for k, v := range r.scopes[i] {
			out[k] = v
		}
	}
	return out
}

// Environment returns the values of the first scope (highest priority).
func (r *Resolver) Environment() map[string]string {
	if len(r.scopes) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(r.scopes[0]))
	for k, v := range r.scopes[0] {
		out[k] = v
	}
	return out
}

func (r *Resolver) generate(name string) string {
	switch name {
	case "$guid", "$randomUUID":
		return uuid()
	case "$timestamp":
		return fmt.Sprintf("%d", time.Now().Unix())
	case "$isoTimestamp":
		return time.Now().UTC().Format(time.RFC3339)
	case "$randomInt":
		return fmt.Sprintf("%d", r.intn(1000000))
	case "$randomBoolean":
		if r.intn(2) == 0 {
			return "true"
		}
		return "false"
	case "$randomEmail":
		return fmt.Sprintf("%s%d@%s", r.pick(firstNames), r.intn(9000)+1000, r.pick(domains))
	case "$randomFirstName":
		return strings.Title(r.pick(firstNames))
	case "$randomLastName":
		return strings.Title(r.pick(lastNames))
	case "$randomFullName":
		return strings.Title(r.pick(firstNames)) + " " + strings.Title(r.pick(lastNames))
	case "$randomUserName":
		return strings.ToLower(r.pick(firstNames)) + fmt.Sprintf("%d", r.intn(1000))
	case "$randomPassword":
		return r.alphaNumeric(12)
	case "$randomAlphaNumeric":
		return r.alphaNumeric(12)
	case "$randomColor":
		return r.pick(colors)
	case "$randomHexColor":
		return fmt.Sprintf("#%06x", r.intn(0xffffff))
	case "$randomIP":
		return fmt.Sprintf("%d.%d.%d.%d", r.intn(256), r.intn(256), r.intn(256), r.intn(256))
	case "$randomPort":
		return fmt.Sprintf("%d", r.intn(60000)+1024)
	case "$randomWord":
		return r.pick(words)
	case "$randomWords":
		return r.pick(words) + " " + r.pick(words) + " " + r.pick(words)
	case "$randomLoremWord":
		return r.pick(words)
	case "$randomLoremSentence":
		return r.pick(words) + " " + r.pick(words) + " " + r.pick(words) + "."
	case "$randomLoremParagraph":
		return strings.Join([]string{
			r.pick(words) + " " + r.pick(words) + " " + r.pick(words) + ".",
			r.pick(words) + " " + r.pick(words) + " " + r.pick(words) + ".",
		}, " ")
	case "$randomCity":
		return r.pick(cities)
	case "$randomCountry":
		return r.pick(countries)
	case "$randomPhoneNumber":
		return fmt.Sprintf("+1%03d%03d%04d", r.intn(900)+100, r.intn(900)+100, r.intn(9000)+1000)
	case "$randomJobTitle":
		return r.pick(jobs)
	case "$randomCompanyName":
		return r.pick(companies)
	case "$randomProductName":
		return r.pick(products)
	case "$randomPrice":
		return fmt.Sprintf("%.2f", float64(r.intn(100000))/100)
	case "$randomDateRecent":
		return time.Now().Add(-time.Duration(r.intn(30)) * 24 * time.Hour).Format("2006-01-02")
	case "$randomDateFuture":
		return time.Now().Add(time.Duration(r.intn(30)) * 24 * time.Hour).Format("2006-01-02")
	case "$randomDatePast":
		return time.Now().Add(-time.Duration(r.intn(3650)) * 24 * time.Hour).Format("2006-01-02")
	case "$randomImageUrl":
		return fmt.Sprintf("https://picsum.photos/seed/%d/200/200", r.intn(10000))
	case "$randomAvatarImage":
		return fmt.Sprintf("https://i.pravatar.cc/150?u=%d", r.intn(10000))
	case "$randomBankAccount":
		return fmt.Sprintf("%010d", r.intn(1000000000))
	default:
		return ""
	}
}

func (r *Resolver) intn(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Intn(n)
}

func (r *Resolver) pick(list []string) string {
	return list[r.intn(len(list))]
}

func (r *Resolver) alphaNumeric(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[r.intn(len(chars))]
	}
	return string(b)
}

func uuid() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst)
}

var (
	firstNames = []string{"james", "mary", "john", "patricia", "robert", "jennifer", "michael", "linda", "william", "elizabeth"}
	lastNames  = []string{"smith", "johnson", "williams", "brown", "jones", "garcia", "miller", "davis", "rodriguez", "martinez"}
	domains    = []string{"example.com", "test.com", "mail.com", "inbox.net"}
	colors     = []string{"red", "green", "blue", "yellow", "purple", "orange", "pink", "cyan", "teal", "magenta"}
	words      = []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet", "kilo", "lima"}
	cities     = []string{"Berlin", "London", "Tokyo", "Paris", "Toronto", "Sydney", "Madrid", "Chicago"}
	countries  = []string{"Germany", "United Kingdom", "Japan", "France", "Canada", "Australia", "Spain", "United States"}
	jobs       = []string{"Software Engineer", "Product Manager", "Designer", "Data Analyst", "DevOps Engineer", "QA Engineer"}
	companies  = []string{"Acme Corp", "Globex", "Initech", "Umbrella", "Hooli", "Stark Industries"}
	products   = []string{"Wireless Mouse", "Mechanical Keyboard", "USB-C Hub", "Monitor Stand", "Laptop Sleeve"}
)
