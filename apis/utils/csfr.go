package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"math/rand"
	"net/textproto"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Option is the only struct that can be used to set Options.
type Option struct {
	F func(o *Options)
}

const (
	csrfSecret     = "csrfSecret"
	csrfSalt       = "csrfSalt"
	csrfToken      = "csrfToken"
	csrfHeaderName = "X-CSRF-TOKEN"

	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// 6 bits to represent a letter index
	letterIdBits = 6
	// All 1-bits as many as letterIdBits
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
)

var (
	errMissingHeader = errors.New("[CSRF] missing csrf token in header")
	errMissingQuery  = errors.New("[CSRF] missing csrf token in query")
	errMissingParam  = errors.New("[CSRF] missing csrf token in param")
	errMissingForm   = errors.New("[CSRF] missing csrf token in form")
	errInvalidToken  = errors.New("[CSRF] invalid token")
)

type CsrfNextHandler func(c *gin.Context) bool

type CsrfExtractorHandler func(c *gin.Context) (string, error)

// Options defines the config for middleware.
type Options struct {
	// Secret used to generate token.
	//
	// Default: csrfSecret
	Secret string

	// Ignored methods will be considered no protection required.
	//
	// Optional. Default: "GET", "HEAD", "OPTIONS", "TRACE"
	IgnoreMethods []string

	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next CsrfNextHandler

	// KeyLookup is a string in the form of "<source>:<key>" that is used
	// to create an Extractor that extracts the token from the request.
	// Possible values:
	// - "header:<name>"
	// - "query:<name>"
	// - "param:<name>"
	// - "form:<name>"
	//
	// Optional. Default: "header:X-CSRF-TOKEN"
	KeyLookup string

	// ErrorFunc is executed when an error is returned from gin.HandlerFunc.
	//
	// Optional. Default: func( c *gin.Context) { panic(c.Errors.Last()) }
	ErrorFunc gin.HandlerFunc

	// Extractor returns the csrf token.
	//
	// If set this will be used in place of an Extractor based on KeyLookup.
	//
	// Optional. Default will create an Extractor based on KeyLookup.
	Extractor CsrfExtractorHandler
}

func (o *Options) Apply(opts []Option) {
	for _, op := range opts {
		op.F(o)
	}
}

// OptionsDefault is the default options.
var OptionsDefault = Options{
	Secret: csrfSecret,
	// Assume that anything not defined as 'safe' by RFC7231 needs protection
	IgnoreMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	Next:          nil,
	KeyLookup:     "header:" + csrfHeaderName,
	ErrorFunc:     func(c *gin.Context) { panic(c.Errors.Last()) },
}

func NewOptions(opts ...Option) *Options {
	options := &Options{
		Secret:        OptionsDefault.Secret,
		IgnoreMethods: OptionsDefault.IgnoreMethods,
		Next:          OptionsDefault.Next,
		KeyLookup:     OptionsDefault.KeyLookup,
		ErrorFunc:     OptionsDefault.ErrorFunc,
	}
	options.Apply(opts)
	return options
}

// WithSecret sets secret.
func WithSecret(secret string) Option {
	return Option{
		F: func(o *Options) {
			o.Secret = secret
		},
	}
}

// WithIgnoredMethods sets methods that do not need to be protected.
func WithIgnoredMethods(methods []string) Option {
	return Option{
		F: func(o *Options) {
			o.IgnoreMethods = methods
		},
	}
}

// WithNext sets whether to skip this middleware.
func WithNext(f CsrfNextHandler) Option {
	return Option{
		F: func(o *Options) {
			o.Next = f
		},
	}
}

// WithKeyLookUp sets a string in the form of "<source>:<key>" that is used
// to create an Extractor that extracts the token from the request.
func WithKeyLookUp(lookup string) Option {
	return Option{
		F: func(o *Options) {
			o.KeyLookup = lookup
		},
	}
}

// WithErrorFunc sets ErrorFunc.
func WithErrorFunc(f gin.HandlerFunc) Option {
	return Option{
		F: func(o *Options) {
			o.ErrorFunc = f
		},
	}
}

// WithExtractor sets extractor.
func WithExtractor(f CsrfExtractorHandler) Option {
	return Option{
		F: func(o *Options) {
			o.Extractor = f
		},
	}
}

// CsrfFromParam returns a function that extracts token from the url param string.
func CsrfFromParam(param string) func(c *gin.Context) (string, error) {
	return func(c *gin.Context) (string, error) {
		token := c.Param(param)
		if token == "" {
			return "", errMissingParam
		}
		return token, nil
	}
}

// CsrfFromForm returns a function that extracts a token from a multipart-form.
func CsrfFromForm(param string) func(c *gin.Context) (string, error) {
	return func(c *gin.Context) (string, error) {
		token := c.Request.FormValue(param)
		if string(token) == "" {
			return "", errMissingForm
		}
		return string(token), nil
	}
}

// CsrfFromHeader returns a function that extracts token from the request header.
func CsrfFromHeader(param string) func(c *gin.Context) (string, error) {
	return func(c *gin.Context) (string, error) {
		token := c.GetHeader(param)
		if string(token) == "" {
			return "", errMissingHeader
		}
		return string(token), nil
	}
}

// CsrfFromQuery returns a function that extracts token from the query string.
func CsrfFromQuery(param string) func(c *gin.Context) (string, error) {
	return func(c *gin.Context) (string, error) {
		token := c.Query(param)
		if token == "" {
			return "", errMissingQuery
		}
		return token, nil
	}
}

// NewCsfrTokenValidation validates CSRF token.
func NewCsfrTokenValidation(opts ...Option) gin.HandlerFunc {
	cfg := NewOptions(opts...)
	selectors := strings.Split(cfg.KeyLookup, ":")

	if len(selectors) != 2 {
		panic(errors.New("[CSRF] KeyLookup must in the form of <source>:<key>"))
	}

	if cfg.Extractor == nil {
		// By default, we extract from a header
		cfg.Extractor = CsrfFromHeader(textproto.CanonicalMIMEHeaderKey(selectors[1]))

		switch selectors[0] {
		case "form":
			cfg.Extractor = CsrfFromForm(selectors[1])
		case "query":
			cfg.Extractor = CsrfFromQuery(selectors[1])
		case "param":
			cfg.Extractor = CsrfFromParam(selectors[1])
		}
	}

	return func(c *gin.Context) {
		// Don't execute middleware if Next returns true
		if cfg.Next != nil && cfg.Next(c) {
			c.Next()
			return
		}

		session := sessions.Default(c)
		c.Set(csrfSecret, cfg.Secret)

		if isIgnored(cfg.IgnoreMethods, c.Request.Method) {
			c.Next()
			return
		}

		token, err := cfg.Extractor(c)
		if err != nil {
			c.Error(err)
			cfg.ErrorFunc(c)
			return
		}

		if tokenize(cfg.Secret, session.ID()) != token {
			c.Error(errInvalidToken)
			cfg.ErrorFunc(c)
			return
		}

		c.Next()
	}
}

// GetToken returns a CSRF token.
func GetToken(c *gin.Context) string {
	session := sessions.Default(c)
	secret := c.MustGet(csrfSecret).(string)

	if t, ok := c.Get(csrfToken); ok {
		return t.(string)
	}

	salt, ok := session.Get(csrfSalt).(string)
	if !ok {
		salt = randStr(16)
		session.Set(csrfSalt, salt)
		session.Save()
	}
	token := tokenize(secret, salt)
	c.Set(csrfToken, token)

	return token
}

// tokenize generates token through secret and salt.
func tokenize(secret, salt string) string {
	h := sha256.New()
	io.WriteString(h, salt+"-"+secret)
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))

	return hash
}

// isIgnored determines whether the method is ignored.
func isIgnored(arr []string, value string) bool {
	ignore := false

	for _, v := range arr {
		if v == value {
			ignore = true
			break
		}
	}

	return ignore
}

var src = rand.NewSource(time.Now().UnixNano())

// randStr generates random string.
func randStr(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters
	for i, cache, remain := n-1, src.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(letters) {
			sb.WriteByte(letters[idx])
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return sb.String()
}
