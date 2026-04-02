package a2ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Resolve resolves a DynamicValue against a DataModel, returning the concrete value.
func Resolve(val DynamicValue, dm *DataModel) (any, error) {
	switch {
	case val.Binding != nil:
		return dm.Get(val.Binding.Path)
	case val.FunctionCall != nil:
		return callBuiltin(val.FunctionCall.Call, val.FunctionCall.Args, dm)
	case val.String != nil:
		return *val.String, nil
	case val.Number != nil:
		return *val.Number, nil
	case val.Bool != nil:
		return *val.Bool, nil
	case val.Array != nil:
		return val.Array, nil
	default:
		return nil, nil
	}
}

// ResolveString resolves a DynamicValue and converts the result to a string.
func ResolveString(val DynamicValue, dm *DataModel) string {
	v, err := Resolve(val, dm)
	if err != nil || v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// ResolveDynamicString resolves a DynamicString against a DataModel.
func ResolveDynamicString(val DynamicString, dm *DataModel) (string, error) {
	switch {
	case val.Binding != nil:
		v, err := dm.Get(val.Binding.Path)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%v", v), nil
	case val.FunctionCall != nil:
		v, err := callBuiltin(val.FunctionCall.Call, val.FunctionCall.Args, dm)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%v", v), nil
	case val.Literal != nil:
		return *val.Literal, nil
	default:
		return "", nil
	}
}

// ResolveDynamicNumber resolves a DynamicNumber against a DataModel.
func ResolveDynamicNumber(val DynamicNumber, dm *DataModel) (float64, error) {
	switch {
	case val.Binding != nil:
		v, err := dm.Get(val.Binding.Path)
		if err != nil {
			return 0, err
		}
		return toFloat(v), nil
	case val.FunctionCall != nil:
		v, err := callBuiltin(val.FunctionCall.Call, val.FunctionCall.Args, dm)
		if err != nil {
			return 0, err
		}
		return toFloat(v), nil
	case val.Literal != nil:
		return *val.Literal, nil
	default:
		return 0, nil
	}
}

// ResolveDynamicBoolean resolves a DynamicBoolean against a DataModel.
func ResolveDynamicBoolean(val DynamicBoolean, dm *DataModel) (bool, error) {
	switch {
	case val.Binding != nil:
		v, err := dm.Get(val.Binding.Path)
		if err != nil {
			return false, err
		}
		return toBool(v), nil
	case val.FunctionCall != nil:
		v, err := callBuiltin(val.FunctionCall.Call, val.FunctionCall.Args, dm)
		if err != nil {
			return false, err
		}
		return toBool(v), nil
	case val.Literal != nil:
		return *val.Literal, nil
	default:
		return false, nil
	}
}

// ResolveDynamicStringList resolves a DynamicStringList against a DataModel.
func ResolveDynamicStringList(val DynamicStringList, dm *DataModel) ([]string, error) {
	switch {
	case val.Binding != nil:
		v, err := dm.Get(val.Binding.Path)
		if err != nil {
			return nil, err
		}
		return toStringSlice(v), nil
	case val.FunctionCall != nil:
		v, err := callBuiltin(val.FunctionCall.Call, val.FunctionCall.Args, dm)
		if err != nil {
			return nil, err
		}
		return toStringSlice(v), nil
	case val.Literal != nil:
		return append([]string(nil), val.Literal...), nil
	default:
		return nil, nil
	}
}

func callBuiltin(name string, args map[string]any, dm *DataModel) (any, error) {
	switch name {
	case "required":
		return builtinRequired(args, dm)
	case "length":
		return builtinLength(args, dm)
	case "numeric":
		return builtinNumeric(args, dm)
	case "email":
		return builtinEmail(args, dm)
	case "regex":
		return builtinRegex(args, dm)
	case "formatString":
		return builtinFormatString(args, dm)
	case "formatNumber":
		return builtinFormatNumber(args, dm)
	case "formatCurrency":
		return builtinFormatCurrency(args, dm)
	case "formatDate":
		return builtinFormatDate(args, dm)
	case "pluralize":
		return builtinPluralize(args, dm)
	case "and":
		return builtinAnd(args, dm)
	case "or":
		return builtinOr(args, dm)
	case "not":
		return builtinNot(args, dm)
	default:
		return nil, fmt.Errorf("unknown function %q", name)
	}
}

func resolveArg(args map[string]any, key string, dm *DataModel) any {
	v, ok := args[key]
	if !ok {
		return nil
	}
	resolved, err := resolveAny(v, dm)
	if err != nil {
		return v
	}
	return resolved
}

func resolveAny(v any, dm *DataModel) (any, error) {
	switch val := v.(type) {
	case DynamicValue:
		return Resolve(val, dm)
	case *DynamicValue:
		if val == nil {
			return nil, nil
		}
		return Resolve(*val, dm)
	case DynamicString:
		return ResolveDynamicString(val, dm)
	case *DynamicString:
		if val == nil {
			return "", nil
		}
		return ResolveDynamicString(*val, dm)
	case DynamicNumber:
		return ResolveDynamicNumber(val, dm)
	case *DynamicNumber:
		if val == nil {
			return 0, nil
		}
		return ResolveDynamicNumber(*val, dm)
	case DynamicBoolean:
		return ResolveDynamicBoolean(val, dm)
	case *DynamicBoolean:
		if val == nil {
			return false, nil
		}
		return ResolveDynamicBoolean(*val, dm)
	case DynamicStringList:
		return ResolveDynamicStringList(val, dm)
	case *DynamicStringList:
		if val == nil {
			return nil, nil
		}
		return ResolveDynamicStringList(*val, dm)
	case DataBinding:
		return dm.Get(val.Path)
	case *DataBinding:
		if val == nil {
			return nil, nil
		}
		return dm.Get(val.Path)
	case string:
		if strings.HasPrefix(val, "/") {
			got, err := dm.Get(val)
			if err == nil {
				return got, nil
			}
		}
		return val, nil
	case map[string]any:
		if path, ok := val["path"].(string); ok {
			return dm.Get(path)
		}
		return val, nil
	default:
		return v, nil
	}
}

func builtinRequired(args map[string]any, dm *DataModel) (bool, error) {
	val := resolveArg(args, "value", dm)
	if val == nil {
		return false, nil
	}
	if s, ok := val.(string); ok {
		return s != "", nil
	}
	if ss, ok := val.([]string); ok {
		return len(ss) > 0, nil
	}
	return true, nil
}

func builtinLength(args map[string]any, dm *DataModel) (any, error) {
	val := resolveArg(args, "value", dm)
	min, hasMin := intArg(args, "min")
	max, hasMax := intArg(args, "max")
	switch v := val.(type) {
	case string:
		n := len(v)
		if hasMin || hasMax {
			return inRange(n, min, hasMin, max, hasMax), nil
		}
		return n, nil
	case []string:
		n := len(v)
		if hasMin || hasMax {
			return inRange(n, min, hasMin, max, hasMax), nil
		}
		return n, nil
	case []any:
		n := len(v)
		if hasMin || hasMax {
			return inRange(n, min, hasMin, max, hasMax), nil
		}
		return n, nil
	case nil:
		if hasMin || hasMax {
			return inRange(0, min, hasMin, max, hasMax), nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("length: unsupported type %T", val)
	}
}

func builtinNumeric(args map[string]any, dm *DataModel) (bool, error) {
	val := resolveArg(args, "value", dm)
	n, ok := numericValue(val)
	if !ok {
		return false, nil
	}
	min, hasMin := floatArg(args, "min")
	max, hasMax := floatArg(args, "max")
	return inRange(n, min, hasMin, max, hasMax), nil
}

func builtinEmail(args map[string]any, dm *DataModel) (bool, error) {
	val := resolveArg(args, "value", dm)
	s, ok := val.(string)
	if !ok {
		return false, nil
	}
	return strings.Contains(s, "@") && strings.Contains(s, "."), nil
}

func builtinRegex(args map[string]any, dm *DataModel) (bool, error) {
	pattern, _ := resolveArg(args, "pattern", dm).(string)
	s, _ := resolveArg(args, "value", dm).(string)
	if pattern == "" {
		return false, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("regex: %w", err)
	}
	return re.MatchString(s), nil
}

func builtinFormatString(args map[string]any, dm *DataModel) (string, error) {
	if value, ok := resolveArg(args, "value", dm).(string); ok && value != "" {
		return interpolateFormatString(value, dm)
	}
	tmpl, _ := resolveArg(args, "template", dm).(string)
	fmtArgs, _ := args["args"].(map[string]any)
	result := tmpl
	for k, v := range fmtArgs {
		resolved, err := resolveAny(v, dm)
		if err != nil {
			resolved = v
		}
		result = strings.ReplaceAll(result, "%{"+k+"}", fmt.Sprintf("%v", resolved))
	}
	return result, nil
}

func builtinFormatNumber(args map[string]any, dm *DataModel) (string, error) {
	n, ok := numericValue(resolveArg(args, "value", dm))
	if !ok {
		return "", nil
	}
	decimals, _ := intArg(args, "decimals")
	grouping := toBool(resolveArg(args, "grouping", dm))
	return formatDecimal(n, decimals, grouping), nil
}

func builtinFormatCurrency(args map[string]any, dm *DataModel) (string, error) {
	currency, _ := resolveArg(args, "currency", dm).(string)
	n, ok := numericValue(resolveArg(args, "value", dm))
	if !ok {
		return "", nil
	}
	decimals, hasDecimals := intArg(args, "decimals")
	if !hasDecimals {
		decimals = 2
	}
	grouping := true
	if _, ok := args["grouping"]; ok {
		grouping = toBool(resolveArg(args, "grouping", dm))
	}
	formatted := formatDecimal(n, decimals, grouping)
	if currency == "" {
		return formatted, nil
	}
	return currency + formatted, nil
}

func builtinFormatDate(args map[string]any, dm *DataModel) (string, error) {
	format, _ := resolveArg(args, "format", dm).(string)
	value := resolveArg(args, "value", dm)
	t, ok := timeValue(value)
	if !ok {
		return "", nil
	}
	if format == "" {
		format = time.RFC3339
	}
	return t.Format(dateLayout(format)), nil
}

func builtinPluralize(args map[string]any, dm *DataModel) (string, error) {
	n, ok := numericValue(resolveArg(args, "value", dm))
	if !ok {
		return "", nil
	}
	category := pluralCategory(n)
	if v, ok := resolveArg(args, category, dm).(string); ok && v != "" {
		return v, nil
	}
	other, _ := resolveArg(args, "other", dm).(string)
	return other, nil
}

func builtinAnd(args map[string]any, dm *DataModel) (bool, error) {
	values, ok := resolveArg(args, "values", dm).([]any)
	if ok && len(values) > 0 {
		for _, value := range values {
			if !toBool(value) {
				return false, nil
			}
		}
		return true, nil
	}
	return toBool(resolveArg(args, "a", dm)) && toBool(resolveArg(args, "b", dm)), nil
}

func builtinOr(args map[string]any, dm *DataModel) (bool, error) {
	values, ok := resolveArg(args, "values", dm).([]any)
	if ok && len(values) > 0 {
		for _, value := range values {
			if toBool(value) {
				return true, nil
			}
		}
		return false, nil
	}
	return toBool(resolveArg(args, "a", dm)) || toBool(resolveArg(args, "b", dm)), nil
}

func builtinNot(args map[string]any, dm *DataModel) (bool, error) {
	a := resolveArg(args, "value", dm)
	if a == nil {
		a = resolveArg(args, "a", dm)
	}
	return !toBool(a), nil
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case float64:
		return val != 0
	case nil:
		return false
	default:
		return true
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func intArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func floatArg(args map[string]any, key string) (float64, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	return numericValue(v)
}

func inRange[T ~int | ~float64](n, min T, hasMin bool, max T, hasMax bool) bool {
	if hasMin && n < min {
		return false
	}
	if hasMax && n > max {
		return false
	}
	return true
}

func interpolateFormatString(format string, dm *DataModel) (string, error) {
	var out strings.Builder
	for i := 0; i < len(format); {
		if format[i] == '\\' && i+2 < len(format) && format[i+1] == '$' && format[i+2] == '{' {
			out.WriteString("${")
			i += 3
			continue
		}
		if format[i] == '$' && i+1 < len(format) && format[i+1] == '{' {
			end := strings.Index(format[i+2:], "}")
			if end < 0 {
				out.WriteString(format[i:])
				break
			}
			expr := strings.TrimSpace(format[i+2 : i+2+end])
			val := expr
			if strings.HasPrefix(expr, "/") {
				resolved, err := dm.Get(expr)
				if err == nil {
					val = fmt.Sprintf("%v", resolved)
				}
			}
			out.WriteString(val)
			i += end + 3
			continue
		}
		out.WriteByte(format[i])
		i++
	}
	return out.String(), nil
}

func formatDecimal(value float64, decimals int, grouping bool) string {
	s := strconv.FormatFloat(value, 'f', decimals, 64)
	if !grouping {
		return s
	}
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = strings.TrimPrefix(s, "-")
	}
	intPart := s
	frac := ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, frac = s[:dot], s[dot:]
	}
	for i := len(intPart) - 3; i > 0; i -= 3 {
		intPart = intPart[:i] + "," + intPart[i:]
	}
	return sign + intPart + frac
}

func timeValue(v any) (time.Time, bool) {
	switch val := v.(type) {
	case string:
		if val == "" {
			return time.Time{}, false
		}
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t, true
		}
		if t, err := time.Parse("2006-01-02", val); err == nil {
			return t, true
		}
	case float64:
		return time.Unix(int64(val), 0).UTC(), true
	case int:
		return time.Unix(int64(val), 0).UTC(), true
	case int64:
		return time.Unix(val, 0).UTC(), true
	}
	return time.Time{}, false
}

func dateLayout(format string) string {
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"yyyy", "2006",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
	)
	return replacer.Replace(format)
}

func pluralCategory(value float64) string {
	switch {
	case value == 0:
		return "zero"
	case value == 1:
		return "one"
	case value == 2:
		return "two"
	case value >= 3 && value <= 4:
		return "few"
	case value >= 5:
		return "many"
	default:
		return "other"
	}
}

func toStringSlice(v any) []string {
	switch ss := v.(type) {
	case []string:
		return append([]string(nil), ss...)
	case []any:
		out := make([]string, 0, len(ss))
		for _, s := range ss {
			out = append(out, fmt.Sprintf("%v", s))
		}
		return out
	default:
		return nil
	}
}
