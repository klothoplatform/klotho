package csharp

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/parseutils"
	"regexp"
	"strings"
)

var optionalLastSegmentPattern = regexp.MustCompile(`(?:^|/)(\{[^?{}]+\?})(/$|$)`)
var defaultLastSegmentPattern = regexp.MustCompile(`(?:^|/)(\{[^{][^}=]*=(?:[^}]|(?:}})+)*[^}]}(}})*)(?:/$|$)`)

// stripOptionalLastSegment strips the last segment of a path if it contains an optional or default param
func stripOptionalLastSegment(routeTemplate string) string {
	match := optionalLastSegmentPattern.FindStringIndex(routeTemplate)
	if match == nil {
		match = defaultLastSegmentPattern.FindStringIndex(routeTemplate)
	}
	if match != nil && !complexSegmentPattern.MatchString(routeTemplate[match[0]:]) {

		routeTemplate = strings.TrimSuffix(routeTemplate, "/")
		routeTemplate = routeTemplate[0:match[0]]
	}
	return routeTemplate
}

var expressParamConversionPattern = regexp.MustCompile(`\{([^:=}?]*)[^}]*}`)

// sanitizeAttributeBasedPath converts ASP.NET Core attribute-based route path parameters to Express syntax,
// but does not perform validation to ensure that the supplied string is a valid ASP.NET Core route.
// As such, there's no expectation of correct output for invalid paths
func sanitizeAttributeBasedPath(path string, area string, controller string, action string) string {
	path = sanitizeRegexConstraints(path)

	// replace params such as {controller=Index}
	specialParamFormat := `(?i)\{\s*%s\s*=\s*%s\s*}`
	path = regexp.MustCompile(fmt.Sprintf(specialParamFormat, "area", area)).ReplaceAllString(path, area)
	path = regexp.MustCompile(fmt.Sprintf(specialParamFormat, "controller", controller)).ReplaceAllString(path, controller)
	path = regexp.MustCompile(fmt.Sprintf(specialParamFormat, "action", action)).ReplaceAllString(path, action)

	// convert to longest possible proxy route when required
	path = sanitizeProxyPath(path)
	path = mergeComplexSegments(path)

	// convert path params to express syntax
	path = expressParamConversionPattern.ReplaceAllString(path, ":$1")

	// replace special tokens
	path = replaceToken(path, "action", action)
	path = replaceToken(path, "area", area)
	path = replaceToken(path, "controller", controller)
	return path
}

// sanitizeConventionalPath converts ASP.NET Core conventional path parameters to Express syntax,
// but does not perform validation to ensure that the supplied string is a valid ASP.net route.
// As such, there's no expectation of correct output for invalid paths
func sanitizeConventionalPath(path string) string {
	path = sanitizeRegexConstraints(path)
	path = sanitizeProxyPath(path)
	path = mergeComplexSegments(path)

	// convert path params to express syntax
	path = expressParamConversionPattern.ReplaceAllString(path, ":$1")
	return path
}

// getTokenPattern returns a compiled pattern for detecting an unescaped ASP.NET Core route token.
func getTokenPattern(token string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(?i)(?:(?:^|[^[])(?:(?:\[\[)*))(?P<token>\[%s])(?:(?:(?:]])*)|$|[^]])(?:[^]]|$)`, token))
}

// tokenPatterns is a set of token detection patterns mapped by their token names
var tokenPatterns = map[string]*regexp.Regexp{
	"action":     getTokenPattern("action"),
	"area":       getTokenPattern("area"),
	"controller": getTokenPattern("controller"),
}

// replaceToken replaces all instances of token in path with the supplied replacement string
func replaceToken(path, token, replacement string) string {
	pattern, ok := tokenPatterns[token]
	if !ok {
		return path
	}
	matches := pattern.FindStringSubmatchIndex(path)
	for ; matches != nil; matches = pattern.FindStringSubmatchIndex(path) {
		path = path[0:matches[2]] + replacement + path[matches[3]:]
	}
	return path
}

// '*' is a literal in ASP.NET Core routes when not the first character in a route param: /path/* -> /path/*, /path/{*slug} -> /path/:rest*
// Routes containing '*' literals are not supported by AWS API Gateway.
var catchAllRegexp = regexp.MustCompile(`\{\*`)

// defaultParamRegex matches default path parameters -- e.g. /{param=default}
// matches the beginning of a path segment, but not the end to enable processing a full path in one pass
var defaultParamRegex = regexp.MustCompile(`(?:^|/)(?P<param>\{[^{][^}=]*=.*?[^}]}(}})*)`)

// sanitizeProxyPath returns the index of the opening curly brace of the path param that should be converted to a proxy param.
func sanitizeProxyPath(path string) string {
	firstProxyParamIndex := -1
	if firstCatchAll := catchAllRegexp.FindStringIndex(path); firstCatchAll != nil {
		firstProxyParamIndex = firstCatchAll[0] + 1 // +1 to avoid stripping the opening "{"
	}
	hasTrailingOptional := false // includes trailing default params
	hasTrailingDefaultChain := false
	firstTrailingOptionalIndex := -1
	if match := optionalLastSegmentPattern.FindStringSubmatchIndex(path); match != nil {
		hasTrailingOptional = true
		firstTrailingOptionalIndex = match[2]
	}
	if matches := defaultParamRegex.FindAllStringSubmatchIndex(path, -1); matches != nil {
		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]
			// match is final segment
			if match[3] == len(path) || (match[3] == len(path)-1 && strings.HasSuffix(path, "/")) {
				firstTrailingOptionalIndex = match[2]
				hasTrailingOptional = true
				continue
			}
			if !hasTrailingOptional {
				break
			}
			// match immediately precedes last detected trailing optional/default
			if match[3] == firstTrailingOptionalIndex-1 {
				firstTrailingOptionalIndex = match[2]
			}
			hasTrailingDefaultChain = true
		}
	}

	if hasTrailingDefaultChain &&
		(firstTrailingOptionalIndex > -1 ||
			(firstProxyParamIndex == -1 &&
				firstTrailingOptionalIndex < firstProxyParamIndex)) {
		firstProxyParamIndex = firstTrailingOptionalIndex + 1
	}

	if firstProxyParamIndex > -1 {
		path = path[0:firstProxyParamIndex]
		path = path[0:strings.LastIndex(path, "{")+1] + "rest*}"
	}
	return path
}

// regexParamStartPattern matches the start of a path param with a regex constraint
var regexParamStartPattern = regexp.MustCompile(`(?i)\{[^:}]+:regex\(`)

// regexStartPattern matches the start of a regex constraint
var regexStartPattern = regexp.MustCompile(`(?i):regex\(`)

// using "\\" as the escape character under the assumption supplied values are normalized to the string literal format
var extractRegexExprs = parseutils.ExpressionExtractor(`\\`, '(', ')')

func extractRegexExpr(input string) string {
	if exprs := extractRegexExprs(input, 1); len(exprs) == 1 {
		return exprs[0]
	}
	return ""
}

func sanitizeRegexConstraints(path string) string {
	sanitized := path
	pStart := regexParamStartPattern.FindStringIndex(sanitized)
	for ; pStart != nil; pStart = regexParamStartPattern.FindStringIndex(sanitized) {
		rStart := regexStartPattern.FindStringIndex(sanitized[pStart[0]:])
		if rStart == nil {
			continue
		}
		regexStart := pStart[0] + rStart[1]
		regexParamEnd := -1
		if regexExpr := extractRegexExpr(sanitized[regexStart-1:]); regexExpr != "" {
			regexParamEnd = (regexStart - 1) + (len(regexExpr) - 1)
		}
		if regexParamEnd != -1 {
			sanitized = sanitized[0:pStart[0]+rStart[0]] + sanitized[regexParamEnd+1:]
		} else {
			return path // prevents infinite loop if the route has any incomplete/invalid regex constraints
		}
	}
	return sanitized
}

var complexSegmentPattern = regexp.MustCompile(`(?P<complex>[^{}/]+(?:[^{}/]*\{[^{}/]+}[^{}/]*?)+?|{[^{}/]+}(?:[^{}/]+|\{[^{}/]+})+)(?:/|$|/$)`)

// mergeComplexSegments replaces complex path segments with uniquely named parameterized segments.
// e.g. /simple/{c}ompl{ex}/{com}.{plex} -> /simple/{complex1}/{complex2}
func mergeComplexSegments(path string) string {
	i := 0
	return complexSegmentPattern.ReplaceAllStringFunc(path, func(complex string) string {
		i++
		segment := fmt.Sprintf("{%s%d}", "complex", i)
		if strings.HasSuffix(complex, "/") {
			segment += "/"
		}
		return segment
	})
}
